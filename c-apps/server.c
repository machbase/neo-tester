/******************************************************************************
 * Simple multi-threaded socket server.
 *
 * Interface:
 *   ./server port [cpu_count] [busyloop] [chunk_count]
 ******************************************************************************/

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>
#include <unistd.h>
#include <time.h>
#include <pthread.h>
#include <netdb.h>
#include <sys/types.h>
#include <sys/socket.h>
#ifdef __linux__
#include <sched.h>
#endif

#define DATA_SIZE 1000
#define STAT_INTERVAL 10000

static int g_chunk_count = 1; /* how many pieces to split each DATA_SIZE recv */
static int g_busyloop = 100000; /* iterations per message for dummy work */

struct thread_args {
    int fd;
    int thread_index;
};

static int send_all(int fd, const unsigned char *buf, size_t len)
{
    size_t sent = 0;
    while (sent < len)
    {
        ssize_t n = send(fd, buf + sent, len - sent, 0);
        if (n < 0)
        {
            if (errno == EINTR)
            {
                continue;
            }
            return -1;
        }
        if (n == 0)
        {
            return -1;
        }
        sent += (size_t)n;
    }
    return 0;
}

static int recv_all(int fd, unsigned char *buf, size_t len)
{
    size_t recvd = 0;
    while (recvd < len)
    {
        ssize_t n = recv(fd, buf + recvd, len - recvd, 0);
        if (n < 0)
        {
            if (errno == EINTR)
            {
                continue;
            }
            return -1;
        }
        if (n == 0)
        {
            return -1;
        }
        recvd += (size_t)n;
    }
    return 0;
}

static int bind_listen(int port, int backlog)
{
    struct addrinfo hints;
    struct addrinfo *res = NULL;
    struct addrinfo *rp = NULL;
    char port_str[16];
    int fd = -1;
    int opt = 1;

    snprintf(port_str, sizeof(port_str), "%d", port);
    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;
    hints.ai_flags = AI_PASSIVE;

    if (getaddrinfo(NULL, port_str, &hints, &res) != 0)
    {
        return -1;
    }

    for (rp = res; rp != NULL; rp = rp->ai_next)
    {
        fd = socket(rp->ai_family, rp->ai_socktype, rp->ai_protocol);
        if (fd < 0)
        {
            continue;
        }
        setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));
        if (bind(fd, rp->ai_addr, rp->ai_addrlen) == 0)
        {
            break;
        }
        close(fd);
        fd = -1;
    }

    freeaddrinfo(res);

    if (fd < 0)
    {
        return -1;
    }

    if (listen(fd, backlog) != 0)
    {
        close(fd);
        return -1;
    }

    return fd;
}

static void *run_thread(void *arg)
{
    struct thread_args *args = (struct thread_args *)arg;
    struct timespec sStartTime;
    struct timespec sEndTime;
    unsigned char buf[DATA_SIZE];
    unsigned long long msg_count = 0;

    const size_t chunk_size = DATA_SIZE / g_chunk_count;
    const size_t remainder = DATA_SIZE % g_chunk_count;

    clock_gettime(CLOCK_MONOTONIC, &sStartTime);

    for (;;)
    {
        int i = 0;
        long sum = (long)args;
        size_t offset = 0;
        int status = 0;

        for (int part = 0; part < g_chunk_count; part++)
        {
            size_t cur = chunk_size + ((size_t)part < remainder ? 1 : 0);
            if (cur == 0)
            {
                continue;
            }
            if (recv_all(args->fd, buf + offset, cur) != 0)
            {
                status = -1;
                break;
            }
            offset += cur;
        }

        if (status != 0)
        {
            break;
        }
        msg_count++;

        for (i = 0; i < g_busyloop; i++) sum++;
        if (STAT_INTERVAL > 0 && (msg_count % STAT_INTERVAL) == 0)
        {
            printf("server thread %d processed %llu messages\n",
                   args->thread_index, msg_count);
        }
        if (send_all(args->fd, buf, DATA_SIZE) != 0)
        {
            break;
        }
    }

    clock_gettime(CLOCK_MONOTONIC, &sEndTime);
    double sElapsedSec = (double)(sEndTime.tv_sec - sStartTime.tv_sec) +
                         (double)(sEndTime.tv_nsec - sStartTime.tv_nsec) / 1000000000.0;
    printf("server thread %d elapsed time: %.6f sec\n",
           args->thread_index, sElapsedSec);

    close(args->fd);
    free(args);
    return NULL;
}

int main(int argc, char **argv)
{
    int sPort = 0;
    int sCpuCount = 8;

    if (argc < 2 || argc > 5)
    {
        fprintf(stderr, "Usage : ./server port [cpu_count] [busyloop] [chunk_count]\n");
        exit(-1);
    }

    sPort = atoi(argv[1]);
    if (argc >= 3)
    {
        sCpuCount = atoi(argv[2]);
    }
    if (argc >= 4)
    {
        g_busyloop = atoi(argv[3]);
    }
    if (argc == 5)
    {
        g_chunk_count = atoi(argv[4]);
    }

    if (sPort <= 0)
    {
        fprintf(stderr, "port must be > 0\n");
        exit(-1);
    }

    if (sCpuCount <= 0)
    {
        sCpuCount = 8;
    }

    if (g_busyloop <= 0)
    {
        g_busyloop = 100000;
    }

    if (g_chunk_count <= 0)
    {
        fprintf(stderr, "chunk_count must be > 0\n");
        exit(-1);
    }

#ifdef __linux__
    long cpu_total = sysconf(_SC_NPROCESSORS_ONLN);
    if (cpu_total > 0 && sCpuCount > cpu_total)
    {
        sCpuCount = (int)cpu_total;
    }
    cpu_set_t set;
    CPU_ZERO(&set);
    for (int i = 0; i < sCpuCount; i++)
    {
        CPU_SET(i, &set);
    }
    if (sched_setaffinity(0, sizeof(set), &set) != 0)
    {
        fprintf(stderr, "sched_setaffinity failed: %s\n", strerror(errno));
    }
#endif

    int listen_fd = bind_listen(sPort, 128);
    if (listen_fd < 0)
    {
        fprintf(stderr, "listen/bind failed\n");
        exit(-1);
    }

    int thread_index = 0;
    for (;;)
    {
        int client_fd = accept(listen_fd, NULL, NULL);
        if (client_fd < 0)
        {
            fprintf(stderr, "accept failed\n");
            close(listen_fd);
            exit(-1);
        }

        struct thread_args *args = calloc(1, sizeof(*args));
        if (args == NULL)
        {
            fprintf(stderr, "memory allocation failed\n");
            close(client_fd);
            continue;
        }
        args->fd = client_fd;
        args->thread_index = thread_index++;

        pthread_t tid;
        if (pthread_create(&tid, NULL, run_thread, args) != 0)
        {
            fprintf(stderr, "pthread_create failed\n");
            close(client_fd);
            free(args);
            close(listen_fd);
            exit(-1);
        }
        pthread_detach(tid);
    }

    close(listen_fd);

    return 0;
}
