/******************************************************************************
 * Copyright of this product 2013-2023,
 * MACHBASE Corporation(or Inc.) or its subsidiaries.
 * All Rights reserved.
 *
 * multi2
 *   - Each thread creates its own connection/statement handle.
 *   - The total number of connections == thread_count, and every thread
 *     uses handles created by other threads (never its own).
 *   - Threads are split into PREPARE threads and EXECUTE/FETCH threads.
 *     Prepare threads repeatedly SQLPrepare on a chosen shared handle; execute
 *     threads wait for a prepared version and then SQLExecute+fetch the same
 *     handle under a mutex.
 *   - Purpose: verify CLI stability when execution threads migrate and use
 *     handles created on different threads.
 ******************************************************************************/

#include <stdio.h>
#include <stdlib.h>
#include <time.h>
#include <pthread.h>
#include <itf.h>
#include <machbase_sqlcli.h>

#define SQL_STR  "select * from tag where name = 'TAG_00' and time between '2017-01-01' and '2017-01-02'"

struct ConnBundle {
    SQLHENV  env;
    SQLHDBC  con;
    SQLHSTMT stmt;

    pthread_mutex_t stmt_lock; /* serialize prepare/execute on this handle */
    pthread_mutex_t prep_lock; /* protect prep_version */
    pthread_cond_t  prep_cond;
    int             prep_version;
};

static struct ConnBundle *gBundles = NULL;
static int gBundleCount = 0;
static pthread_barrier_t gBarrier;

static void printErrorBundle(SQLHENV aEnv, SQLHDBC aCon, SQLHSTMT aStmt, const char *aMsg)
{
    SQLINTEGER      sNativeError;
    SQLCHAR         sErrorMsg[SQL_MAX_MESSAGE_LENGTH + 1];
    SQLCHAR         sSqlState[SQL_SQLSTATE_SIZE + 1];
    SQLSMALLINT     sMsgLength;

    if (aMsg != NULL)
    {
        printf("%s\n", aMsg);
    }

    if (SQLError(aEnv, aCon, aStmt, sSqlState, &sNativeError,
                 sErrorMsg, SQL_MAX_MESSAGE_LENGTH, &sMsgLength) == SQL_SUCCESS)
    {
        printf("SQLSTATE-[%s], Machbase-[%d][%s]\n", sSqlState, sNativeError, sErrorMsg);
    }
}

static void outErrorBundle(const char *aMsg, struct ConnBundle *b, SQLHSTMT aStmt)
{
    printf("ERROR : (%s)\n", aMsg);
    printErrorBundle(b->env, b->con, aStmt, NULL);
    exit(-1);
}

static void db_connect_bundle(struct ConnBundle *b, const char *sHost, unsigned int sPort)
{
    char sConStr[1024];
    SQLINTEGER sErrorNo;
    short sMsgLength;
    char sErrorMsg[1024];

    if (SQL_ERROR == SQLAllocEnv(&b->env))
    {
        printf("SQLAllocEnv error!!\n");
        exit(1);
    }

    if (SQL_ERROR == SQLAllocConnect(b->env, &b->con))
    {
        printf("SQLAllocConnect error!!\n");
        SQLFreeEnv(b->env);
        exit(1);
    }

    snprintf(sConStr, sizeof(sConStr),
             "DSN=%s;UID=SYS;PWD=MANAGER;CONNTYPE=1;PORT_NO=%d",
             sHost, sPort);

    if (SQL_ERROR == SQLDriverConnect(b->con, NULL,
                                      (SQLCHAR *)sConStr, SQL_NTS,
                                      NULL, 0, NULL, SQL_DRIVER_NOPROMPT))
    {
        printf("connection error\n");

        if (SQL_SUCCESS == SQLError(b->env, b->con, NULL, NULL, &sErrorNo,
                                    (SQLCHAR *)sErrorMsg, 1024, &sMsgLength))
        {
            printf(" rCM_-%d : %s\n", sErrorNo, sErrorMsg);
        }
        SQLFreeEnv(b->env);
        exit(1);
    }
}

static void db_disconnect_bundle(struct ConnBundle *b)
{
    SQLINTEGER sErrorNo;
    short sMsgLength;
    char sErrorMsg[1024];

    if (SQL_ERROR == SQLDisconnect(b->con))
    {
        printf("disconnect error\n");

        if (SQL_SUCCESS == SQLError(b->env, b->con, NULL, NULL, &sErrorNo,
                                    (SQLCHAR *)sErrorMsg, 1024, &sMsgLength))
        {
            printf(" rCM_-%d : %s\n", sErrorNo, sErrorMsg);
        }
    }
    SQLFreeConnect(b->con);
    SQLFreeEnv(b->env);
}

static int prepare_stmt(struct ConnBundle *b)
{
    if (SQLPrepare(b->stmt, (SQLCHAR *)SQL_STR, SQL_NTS) != SQL_SUCCESS)
    {
        printErrorBundle(b->env, b->con, b->stmt, "SQLPrepare Error");
        return -1;
    }
    return 0;
}

static int execute_fetch_stmt(struct ConnBundle *b, int aPrint)
{
    SQLLEN sIdLen = 0;
    SQLLEN sValueLen = 0;
    SQLLEN sRegDateLen = 0;

    char sId[33];
    double sValue;
    SQL_TIMESTAMP_STRUCT sRegDate;

    if (SQLExecute(b->stmt) != SQL_SUCCESS)
    {
        printErrorBundle(b->env, b->con, b->stmt, "SQLExecute Error");
        goto error;
    }

    if (SQLBindCol(b->stmt, 1, SQL_C_CHAR, sId, sizeof(sId), &sIdLen) != SQL_SUCCESS)
    {
        printErrorBundle(b->env, b->con, b->stmt, "SQLBindCol 1 Error");
        goto error;
    }

    if (SQLBindCol(b->stmt, 2, SQL_C_TYPE_TIMESTAMP, &sRegDate, 0, &sRegDateLen) != SQL_SUCCESS)
    {
        printErrorBundle(b->env, b->con, b->stmt, "SQLBindCol 2 Error");
        goto error;
    }

    if (SQLBindCol(b->stmt, 3, SQL_C_DOUBLE, &sValue, 0, &sValueLen) != SQL_SUCCESS)
    {
        printErrorBundle(b->env, b->con, b->stmt, "SQLBindCol 3 Error");
        goto error;
    }

    if (aPrint != 0)
    {
        printf("--------------------------------------------------------------------------------\n");
        printf("%-33s%-33s%-33s\n", "NAME", "TIME", "VALUE");
        printf("--------------------------------------------------------------------------------\n");
    }

    while (SQLFetch(b->stmt) == SQL_SUCCESS)
    {
        if (aPrint != 0)
        {
            printf("%-33s", sId);
            printf("%d-%02d-%02d %02d:%02d:%02d 000:000:000 ",
                   sRegDate.year, sRegDate.month, sRegDate.day,
                   sRegDate.hour, sRegDate.minute, sRegDate.second);
            printf("%-.2lf", sValue);
            printf("\n");
        }
    }

    if (SQLFreeStmt(b->stmt, SQL_CLOSE) != SQL_SUCCESS)
    {
        printErrorBundle(b->env, b->con, b->stmt, "SQLFreeStmt Error");
        goto error;
    }

    return 0;

error:
    return -1;
}

struct thread_args {
    char *host;
    int port;
    int test_num;
    int thread_index;
    int print_rows;
    int prep_threads;
    int total_threads;
};

static int pick_target(int self_idx, int iter, int total)
{
    /* choose a handle not created by this thread */
    return (self_idx + iter + 1) % total;
}

void *run_thread(void *arg)
{
    struct thread_args *args = (struct thread_args *)arg;
    int is_prep = (args->thread_index < args->prep_threads);
    struct ConnBundle *mine = &gBundles[args->thread_index];

    db_connect_bundle(mine, args->host, args->port);
    if (SQLAllocStmt(mine->con, &mine->stmt) == SQL_ERROR)
    {
        outErrorBundle("AllocStmt", mine, mine->stmt);
    }

    /* Wait until all connections/statements are ready. */
    pthread_barrier_wait(&gBarrier);

    int *local_versions = calloc((size_t)gBundleCount, sizeof(int));
    if (local_versions == NULL)
    {
        fprintf(stderr, "memory allocation failed\n");
        return NULL;
    }

    struct timespec sStartTime;
    struct timespec sEndTime;
    clock_gettime(CLOCK_MONOTONIC, &sStartTime);

    for (int i = 0; i < args->test_num; i++)
    {
        int target = pick_target(args->thread_index, i, gBundleCount);
        struct ConnBundle *b = &gBundles[target];

        if (is_prep)
        {
            pthread_mutex_lock(&b->stmt_lock);
            if (prepare_stmt(b) != 0)
            {
                pthread_mutex_unlock(&b->stmt_lock);
                outErrorBundle("Prepare", b, b->stmt);
            }
            pthread_mutex_unlock(&b->stmt_lock);

            pthread_mutex_lock(&b->prep_lock);
            b->prep_version++;
            pthread_cond_broadcast(&b->prep_cond);
            pthread_mutex_unlock(&b->prep_lock);
        }
        else
        {
            pthread_mutex_lock(&b->prep_lock);
            while (b->prep_version <= local_versions[target])
            {
                pthread_cond_wait(&b->prep_cond, &b->prep_lock);
            }
            local_versions[target] = b->prep_version;
            pthread_mutex_unlock(&b->prep_lock);

            pthread_mutex_lock(&b->stmt_lock);
            execute_fetch_stmt(b, args->print_rows);
            pthread_mutex_unlock(&b->stmt_lock);
        }
    }

    clock_gettime(CLOCK_MONOTONIC, &sEndTime);
    double sElapsedSec = (double)(sEndTime.tv_sec - sStartTime.tv_sec) +
                         (double)(sEndTime.tv_nsec - sStartTime.tv_nsec) / 1000000000.0;

    free(local_versions);

    /* Ensure all threads finished using shared handles before cleanup. */
    pthread_barrier_wait(&gBarrier);

    if (SQL_ERROR == SQLFreeStmt(mine->stmt, SQL_DROP))
    {
        outErrorBundle("FreeStmt", mine, mine->stmt);
    }
    db_disconnect_bundle(mine);

    pthread_barrier_wait(&gBarrier); /* wait for everyone to disconnect */

    printf("thread %d elapsed time between first execute and barrier: %.6f sec\n",
           args->thread_index, sElapsedSec);

    return NULL;
}

int main(int argc, char **argv)
{
    char * sHost    = NULL;
    int    sPort    = 0;
    int    sTestNum = 0;
    int    sThreadCount = 0;
    int    sPrintRows = 0;
    int    sPrepThreads = 0;

    if (argc != 6)
    {
        fprintf(stderr, "Usage : ./multi2 host port test_num thread_count(>=2) print_rows\n");
        exit(-1);
    }
    else
    {
        switch (argc)
        {
            case 6:
                sPrintRows = atoi(argv[5]);
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

    if (sThreadCount < 2)
    {
        fprintf(stderr, "thread_count must be >= 2\n");
        exit(-1);
    }

    /* Split threads into prepare and execute roles. Ensure at least one of each. */
    sPrepThreads = sThreadCount / 2;
    if (sPrepThreads == 0)
    {
        sPrepThreads = 1;
    }
    int sExecThreads = sThreadCount - sPrepThreads;
    if (sExecThreads == 0)
    {
        sExecThreads = 1;
        sPrepThreads = sThreadCount - 1;
    }

    gBundleCount = sThreadCount;
    gBundles = calloc((size_t)gBundleCount, sizeof(*gBundles));
    if (gBundles == NULL)
    {
        fprintf(stderr, "memory allocation failed\n");
        exit(-1);
    }

    for (int i = 0; i < gBundleCount; i++)
    {
        pthread_mutex_init(&gBundles[i].stmt_lock, NULL);
        pthread_mutex_init(&gBundles[i].prep_lock, NULL);
        pthread_cond_init(&gBundles[i].prep_cond, NULL);
        gBundles[i].prep_version = 0;
        gBundles[i].env = SQL_NULL_HENV;
        gBundles[i].con = SQL_NULL_HDBC;
        gBundles[i].stmt = SQL_NULL_HSTMT;
    }

    if (pthread_barrier_init(&gBarrier, NULL, (unsigned int)sThreadCount) != 0)
    {
        fprintf(stderr, "failed to init barrier\n");
        free(gBundles);
        exit(-1);
    }

    pthread_t *threads = calloc((size_t)sThreadCount, sizeof(*threads));
    struct thread_args *args = calloc((size_t)sThreadCount, sizeof(*args));

    if (threads == NULL || args == NULL)
    {
        fprintf(stderr, "memory allocation failed\n");
        free(threads);
        free(args);
        free(gBundles);
        exit(-1);
    }

    for (int i = 0; i < sThreadCount; i++)
    {
        args[i].host = sHost;
        args[i].port = sPort;
        args[i].test_num = sTestNum;
        args[i].thread_index = i;
        args[i].print_rows = sPrintRows;
        args[i].prep_threads = sPrepThreads;
        args[i].total_threads = sThreadCount;

        if (pthread_create(&threads[i], NULL, run_thread, &args[i]) != 0)
        {
            fprintf(stderr, "pthread_create failed\n");
            free(threads);
            free(args);
            free(gBundles);
            exit(-1);
        }
    }

    for (int i = 0; i < sThreadCount; i++)
    {
        pthread_join(threads[i], NULL);
    }

    pthread_barrier_destroy(&gBarrier);

    free(threads);
    free(args);
    free(gBundles);

    return 0;
}

