/******************************************************************************
 * Simple MySQL benchmark modeled after multi.c
 * - Spawns multiple threads.
 * - Each iteration inserts one row into tmp table and reads it back via
 *   a direct mysql_query() call.
 *
 * Build example:
 *   gcc -o mysql mysql.c -lmysqlclient -lpthread
 *
 * Optional:
 *   export MYSQL_DB=your_db   (defaults to "test")
 ******************************************************************************/

#include <mysql/mysql.h>
#include <stdbool.h>
#include <pthread.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

#define DEFAULT_DB "test"
#define TMP_TABLE  "tmp"

static __thread MYSQL *gCon;

static const char *get_db_name(void)
{
    const char *db = getenv("MYSQL_DB");
    return (db != NULL && db[0] != '\0') ? db : DEFAULT_DB;
}

static void die(const char *msg, MYSQL *conn)
{
    if (conn != NULL)
    {
        fprintf(stderr, "%s: (%u) %s\n", msg, mysql_errno(conn), mysql_error(conn));
    }
    else
    {
        fprintf(stderr, "%s\n", msg);
    }
    exit(EXIT_FAILURE);
}

static void db_connect(const char *host, unsigned int port)
{
    gCon = mysql_init(NULL);
    if (gCon == NULL)
    {
        die("mysql_init failed", gCon);
    }

    bool reconnect = true;
    mysql_options(gCon, MYSQL_OPT_RECONNECT, &reconnect);

    if (mysql_real_connect(gCon, host, "root", "root", get_db_name(), port, NULL, 0) == NULL)
    {
        die("mysql_real_connect failed", gCon);
    }
}

static void db_disconnect(void)
{
    if (gCon != NULL)
    {
        mysql_close(gCon);
        gCon = NULL;
    }
}

static void ensure_schema(const char *host, unsigned int port)
{
    const char *db = get_db_name();
    MYSQL *conn = mysql_init(NULL);
    if (conn == NULL)
    {
        die("mysql_init failed (schema)", conn);
    }

    /* Connect without selecting a DB so we can create it if missing. */
    if (mysql_real_connect(conn, host, "root", "root", NULL, port, NULL, 0) == NULL)
    {
        die("mysql_real_connect failed (schema)", conn);
    }

    char create_db_sql[128];
    snprintf(create_db_sql, sizeof(create_db_sql), "CREATE DATABASE IF NOT EXISTS `%s`", db);
    if (mysql_query(conn, create_db_sql) != 0)
    {
        die("failed to create database", conn);
    }

    if (mysql_select_db(conn, db) != 0)
    {
        die("mysql_select_db failed", conn);
    }

    const char *ddl =
        "CREATE TABLE IF NOT EXISTS " TMP_TABLE " ("
        " id BIGINT AUTO_INCREMENT PRIMARY KEY,"
        " val VARCHAR(64) NOT NULL,"
        " created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP"
        ")";

    if (mysql_query(conn, ddl) != 0)
    {
        die("failed to create tmp table", conn);
    }

    /* Start from a clean slate for consistent timings. */
    if (mysql_query(conn, "TRUNCATE TABLE " TMP_TABLE) != 0)
    {
        die("failed to truncate tmp table", conn);
    }

    mysql_close(conn);
}

static int directExecute2(int print_rows)
{
    /* const char *insert_sql = "INSERT INTO " TMP_TABLE " (val) VALUES ('hello')"; */

    /* if (mysql_query(gCon, insert_sql) != 0) */
    /* { */
    /*     fprintf(stderr, "insert failed: (%u) %s\n", mysql_errno(gCon), mysql_error(gCon)); */
    /*     return -1; */
    /* } */

    /* unsigned long long inserted_id = mysql_insert_id(gCon); */

    char select_sql[128];
    snprintf(select_sql, sizeof(select_sql),
             "SELECT id, val, created_at FROM " TMP_TABLE " WHERE id=1 limit 1;");

    if (print_rows != 0)
    {
        fprintf(stderr, "query : %s\n", select_sql);
    }
    if (mysql_query(gCon, select_sql) != 0)
    {
        fprintf(stderr, "select failed: (%u) %s\n", mysql_errno(gCon), mysql_error(gCon));
        return -1;
    }

    MYSQL_RES *res = mysql_store_result(gCon);
    if (res == NULL)
    {
        fprintf(stderr, "store_result failed: (%u) %s\n", mysql_errno(gCon), mysql_error(gCon));
        return -1;
    }

    MYSQL_ROW row = mysql_fetch_row(res);
    if (row != NULL && print_rows != 0)
    {
        printf("%s\t%s\t%s\n",
               row[0] ? row[0] : "NULL",
               row[1] ? row[1] : "NULL",
               row[2] ? row[2] : "NULL");
    }

    mysql_free_result(res);
    return 0;
}

struct thread_args
{
    const char *host;
    int port;
    int test_num;
    int thread_index;
    int print_rows;
};

static void *run_thread(void *arg)
{
    struct thread_args *args = (struct thread_args *)arg;
    struct timespec sStartTime;
    struct timespec sEndTime;

    mysql_thread_init();
    db_connect(args->host, (unsigned int)args->port);

    clock_gettime(CLOCK_MONOTONIC, &sStartTime);

    for (int i = 0; i < args->test_num; i++)
    {
        if (directExecute2(args->print_rows) != 0)
        {
            break;
        }
    }

    clock_gettime(CLOCK_MONOTONIC, &sEndTime);

    double sElapsedSec = (double)(sEndTime.tv_sec - sStartTime.tv_sec) +
                         (double)(sEndTime.tv_nsec - sStartTime.tv_nsec) / 1000000000.0;

    printf("thread %d elapsed time between connect and disconnect: %.6f sec\n",
           args->thread_index, sElapsedSec);

    db_disconnect();
    mysql_thread_end();
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
        fprintf(stderr, "Usage : ./mysql host port test_num thread_count print_rows\n");
        return EXIT_FAILURE;
    }
    else
    {
        sHost = argv[1];
        sPort = atoi(argv[2]);
        sTestNum = atoi(argv[3]);
        sThreadCount = atoi(argv[4]);
        sPrintRows = atoi(argv[5]);
    }

    if (sThreadCount <= 0)
    {
        fprintf(stderr, "thread_count must be > 0\n");
        return EXIT_FAILURE;
    }

    if (mysql_library_init(0, NULL, NULL) != 0)
    {
        fprintf(stderr, "mysql_library_init failed\n");
        return EXIT_FAILURE;
    }

    /* Prepare schema once before threads start. */
    /* ensure_schema(sHost, (unsigned int)sPort); */

    pthread_t *threads = calloc((size_t)sThreadCount, sizeof(*threads));
    struct thread_args *args = calloc((size_t)sThreadCount, sizeof(*args));

    if (threads == NULL || args == NULL)
    {
        fprintf(stderr, "memory allocation failed\n");
        free(threads);
        free(args);
        mysql_library_end();
        return EXIT_FAILURE;
    }

    struct timespec start_ts;
    struct timespec end_ts;

    clock_gettime(CLOCK_MONOTONIC, &start_ts);

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
            mysql_library_end();
            return EXIT_FAILURE;
        }
    }

    for (int i = 0; i < sThreadCount; i++)
    {
        pthread_join(threads[i], NULL);
    }

    clock_gettime(CLOCK_MONOTONIC, &end_ts);

    double total_elapsed = (double)(end_ts.tv_sec - start_ts.tv_sec) +
                           (double)(end_ts.tv_nsec - start_ts.tv_nsec) / 1000000000.0;
    long long total_requests = (long long)sTestNum * (long long)sThreadCount;
    double rps = total_elapsed > 0.0 ? (double)total_requests / total_elapsed : 0.0;

    printf("rdb total elapsed time: %.6f sec\n", total_elapsed);
    printf("rdb total throughput: %.2f req/sec (%lld requests)\n", rps, total_requests);

    free(threads);
    free(args);

    mysql_library_end();
    return EXIT_SUCCESS;
}
