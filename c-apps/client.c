/******************************************************************************
 * Simple multi-threaded socket client.
 *
 * Interface is compatible with multi.c:
 *   ./client host port test_num thread_count print_rows
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

#define DATA_SIZE 1000

static int gPrtRow = 0;

struct thread_args {
    const char *host;
    int port;
    int test_num;
    int thread_index;
    int print_rows;
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

static int connect_to_host(const char *host, int port)
{
    struct addrinfo hints;
    struct addrinfo *res = NULL;
    struct addrinfo *rp = NULL;
    char port_str[16];
    int fd = -1;

    snprintf(port_str, sizeof(port_str), "%d", port);
    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;

    if (getaddrinfo(host, port_str, &hints, &res) != 0)
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
        if (connect(fd, rp->ai_addr, rp->ai_addrlen) == 0)
        {
            break;
        }
        close(fd);
        fd = -1;
    }

    freeaddrinfo(res);
    return fd;
}

static void fill_payload(unsigned char *buf, int thread_index, int iter)
{
    memset(buf, 0, DATA_SIZE);
    snprintf((char *)buf, DATA_SIZE, "thread=%d iter=%d", thread_index, iter);
}

static void *run_thread(void *arg)
{
    struct thread_args *args = (struct thread_args *)arg;
    struct timespec sStartTime;
    struct timespec sEndTime;
    unsigned char sendbuf[DATA_SIZE];
    unsigned char recvbuf[DATA_SIZE];

    int fd = connect_to_host(args->host, args->port);
    if (fd < 0)
    {
        fprintf(stderr, "thread %d connect failed\n", args->thread_index);
        return NULL;
    }

    clock_gettime(CLOCK_MONOTONIC, &sStartTime);

    for (int i = 0; i < args->test_num; i++)
    {
        fill_payload(sendbuf, args->thread_index, i);
        if (send_all(fd, sendbuf, DATA_SIZE) != 0)
        {
            fprintf(stderr, "thread %d send failed\n", args->thread_index);
            break;
        }
        if (recv_all(fd, recvbuf, DATA_SIZE) != 0)
        {
            fprintf(stderr, "thread %d recv failed\n", args->thread_index);
            break;
        }
        if (args->print_rows != 0)
        {
            printf("client thread %d iter %d recv: %.32s\n",
                   args->thread_index, i, recvbuf);
        }
    }

    clock_gettime(CLOCK_MONOTONIC, &sEndTime);
    double sElapsedSec = (double)(sEndTime.tv_sec - sStartTime.tv_sec) +
                         (double)(sEndTime.tv_nsec - sStartTime.tv_nsec) / 1000000000.0;
    printf("client thread %d elapsed time: %.6f sec\n",
           args->thread_index, sElapsedSec);

    close(fd);
    return NULL;
}

int main(int argc, char **argv)
{
    const char *sHost = NULL;
    int sPort = 0;
    int sTestNum = 0;
    int sThreadCount = 0;
    int sPrintRows = 0;

    if (argc != 6)
    {
        fprintf(stderr, "Usage : ./client host port test_num thread_count print_rows\n");
        exit(-1);
    }
    else
    {
        switch (argc)
        {
            case 6:
                gPrtRow = sPrintRows = atoi(argv[5]);
            case 5:
                sThreadCount = atoi(argv[4]);
            case 4:
                sTestNum = atoi(argv[3]);
            case 3:
                sPort = atoi(argv[2]);
            case 2:
                sHost = argv[1];
                break;
            default:
                break;
        }
    }

    if (sThreadCount <= 0)
    {
        fprintf(stderr, "thread_count must be > 0\n");
        exit(-1);
    }

    pthread_t *threads = calloc((size_t)sThreadCount, sizeof(*threads));
    struct thread_args *args = calloc((size_t)sThreadCount, sizeof(*args));

    if (threads == NULL || args == NULL)
    {
        fprintf(stderr, "memory allocation failed\n");
        free(threads);
        free(args);
        exit(-1);
    }

    for (int i = 0; i < sThreadCount; i++)
    {
        args[i].host = sHost;
        args[i].port = sPort;
        args[i].test_num = sTestNum;
        args[i].thread_index = i;
        args[i].print_rows = sPrintRows;

        if (pthread_create(&threads[i], NULL, run_thread, &args[i]) != 0)
        {
            fprintf(stderr, "pthread_create failed\n");
            free(threads);
            free(args);
            exit(-1);
        }
    }

    for (int i = 0; i < sThreadCount; i++)
    {
        pthread_join(threads[i], NULL);
    }

    free(threads);
    free(args);

    return 0;
}
