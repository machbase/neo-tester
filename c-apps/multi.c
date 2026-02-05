/******************************************************************************
 * Copyright of this product 2013-2023,
 * MACHBASE Corporation(or Inc.) or its subsidiaries.
 * All Rights reserved.
 ******************************************************************************/

#include <stdio.h>
#include <stdlib.h>
#include <time.h>
#include <pthread.h>
#include <string.h>
#include <itf.h>
#include <machbase_sqlcli.h>

#define TEST_LOGTABLE "test_logtbl"

static __thread SQLHENV gEnv;
static __thread SQLHDBC gCon;

void db_connect(char * sHost, unsigned int sPort)
{
    char   sConStr[1024];
    SQLINTEGER   sErrorNo;
    short sMsgLength;
    char   sErrorMsg[1024];

    if (SQL_ERROR == SQLAllocEnv(&gEnv))
    {
        printf("SQLAllocEnv error!!\n");
        exit(1);
    }

    if (SQL_ERROR == SQLAllocConnect(gEnv, &gCon))
    {
        printf("SQLAllocConnect error!!\n");
        SQLFreeEnv(gEnv);
        exit(1);
    }

    snprintf(sConStr,
             sizeof(sConStr),
             "DSN=%s;UID=SYS;PWD=MANAGER;CONNTYPE=1;PORT_NO=%d",
             sHost,
             sPort);

    if (SQL_ERROR == SQLDriverConnect(gCon, NULL,
                                      (SQLCHAR *)sConStr,
                                      SQL_NTS,
                                      NULL, 0, NULL,
                                      SQL_DRIVER_NOPROMPT))
    {
        printf("connection error\n");

        if (SQL_SUCCESS == SQLError(gEnv, gCon, NULL, NULL, &sErrorNo,
                                    (SQLCHAR *)sErrorMsg, 1024, &sMsgLength))
        {
            printf(" rCM_-%d : %s\n", sErrorNo, sErrorMsg);
        }
        SQLFreeEnv(gEnv);
        exit(1);
    }
    printf("connected ... \n");
}

void db_disconnect()
{
    SQLINTEGER   sErrorNo;
    short sMsgLength;
    char   sErrorMsg[1024];

    if (SQL_ERROR == SQLDisconnect(gCon))
    {
        printf("disconnect error\n");

        if (SQL_SUCCESS == SQLError(gEnv, gCon, NULL, NULL, &sErrorNo,
                                    (SQLCHAR *)sErrorMsg, 1024, &sMsgLength))
        {
            printf(" rCM_-%d : %s\n", sErrorNo, sErrorMsg);
        }
    }
    SQLFreeConnect(gCon);
    SQLFreeEnv(gEnv);
}

void outError(const char *aMsg, SQLHSTMT aStmt)
{
    SQLINTEGER sErrorNo;
    short sMsgLength;
    char sErrorMsg[1024];

    printf("ERROR : (%s) \n", aMsg);

    if (SQL_SUCCESS == SQLError(gEnv, gCon, aStmt, NULL, &sErrorNo,
                                (SQLCHAR *)sErrorMsg, 1024, &sMsgLength))
    {
        printf(" mach-%05d : %s\n", sErrorNo, sErrorMsg);
    }

    exit(-1);
}

void printError(SQLHENV aEnv, SQLHDBC aCon, SQLHSTMT aStmt, char *aMsg)
{
    SQLINTEGER      sNativeError;
    SQLCHAR         sErrorMsg[SQL_MAX_MESSAGE_LENGTH + 1];
    SQLCHAR         sSqlState[SQL_SQLSTATE_SIZE + 1];
    SQLSMALLINT     sMsgLength;

    if( aMsg != NULL )
    {
        printf("%s\n", aMsg);
    }

    if( SQLError(aEnv, aCon, aStmt, sSqlState, &sNativeError,
        sErrorMsg, SQL_MAX_MESSAGE_LENGTH, &sMsgLength) == SQL_SUCCESS )
    {
        printf("SQLSTATE-[%s], Machbase-[%d][%s]\n", sSqlState, sNativeError, sErrorMsg);
    }
}

#define SQL_STR  "select * from tag where name = 'TAG_00' and time between '2017-01-01' and '2017-01-02'"
//#define SQL_STR  "select * from v$version limit 1"
//#define SQL_STR  "commit"

int directExecute2(SQLHSTMT aStmt, int aPrint)
{
    const char *sSQL = SQL_STR;

    SQLLEN      sIdLen      = 0;
    SQLLEN      sValueLen   = 0;
    SQLLEN      sRegDateLen = 0;

    char                 sId[33];
    double               sValue;
    SQL_TIMESTAMP_STRUCT sRegDate;

    if( SQLExecDirect(aStmt, (SQLCHAR *)sSQL, SQL_NTS) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, aStmt, "SQLExecDirect Error");
        goto error;
    }

    if( SQLBindCol(aStmt, 1, SQL_C_CHAR, sId, sizeof(sId), &sIdLen) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, aStmt, "SQLBindCol 1 Error");
        goto error;
    }

    if( SQLBindCol(aStmt, 2, SQL_C_TYPE_TIMESTAMP, &sRegDate, 0, &sRegDateLen) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, aStmt, "SQLBindCol 2 Error");
        goto error;
    }

    if( SQLBindCol(aStmt, 3, SQL_C_DOUBLE, &sValue, 0, &sValueLen) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, aStmt, "SQLBindCol 3 Error");
        goto error;
    }

    if (aPrint != 0)
    {
        printf("--------------------------------------------------------------------------------\n");
        printf("%-33s%-33s%-33s\n","NAME","TIME","VALUE");
        printf("--------------------------------------------------------------------------------\n");
    }

    while( SQLFetch(aStmt) == SQL_SUCCESS )
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

    /* if( SQLFreeStmt(aStmt, SQL_CLOSE) != SQL_SUCCESS ) */
    /* { */
    /*     printError(gEnv, gCon, aStmt, "SQLFreeStmt Error"); */
    /*     goto error; */
    /* } */

    return 0;

error:
    return -1;
}

int prepareExecute(SQLHSTMT aStmt, int aPrint)
{
    SQLLEN      sIdLen      = 0;
    SQLLEN      sValueLen   = 0;
    SQLLEN      sRegDateLen = 0;

    char                 sId[33];
    double               sValue;
    SQL_TIMESTAMP_STRUCT sRegDate;

    if( SQLExecute(aStmt) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, aStmt, "SQLExecute Error");
        goto error;
    }

    if( SQLBindCol(aStmt, 1, SQL_C_CHAR, sId, sizeof(sId), &sIdLen) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, aStmt, "SQLBindCol 1 Error");
        goto error;
    }

    if( SQLBindCol(aStmt, 2, SQL_C_TYPE_TIMESTAMP, &sRegDate, 0, &sRegDateLen) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, aStmt, "SQLBindCol 2 Error");
        goto error;
    }

    if( SQLBindCol(aStmt, 3, SQL_C_DOUBLE, &sValue, 0, &sValueLen) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, aStmt, "SQLBindCol 3 Error");
        goto error;
    }

    if (aPrint != 0)
    {
        printf("--------------------------------------------------------------------------------\n");
        printf("%-33s%-33s%-33s\n","NAME","TIME","VALUE");
        printf("--------------------------------------------------------------------------------\n");
    }

    while( SQLFetch(aStmt) == SQL_SUCCESS )
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

    /* if( SQLFreeStmt(aStmt, SQL_CLOSE) != SQL_SUCCESS ) */
    /* { */
    /*     printError(gEnv, gCon, aStmt, "SQLFreeStmt(SQL_CLOSE) Error"); */
    /*     goto error; */
    /* } */

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
    int use_prepare;
};

void *run_thread(void *arg)
{
    struct thread_args *args = (struct thread_args *)arg;
    struct timespec sStartTime;
    struct timespec sEndTime;
    SQLHSTMT sStmt;

    db_connect(args->host, args->port);
    clock_gettime(CLOCK_MONOTONIC, &sStartTime);

    if (SQLAllocStmt(gCon, &sStmt) == SQL_ERROR)
    {
        outError("AllocStmt", sStmt);
    }

    if (args->use_prepare)
    {
        if (SQLPrepare(sStmt, (SQLCHAR *)SQL_STR, SQL_NTS) == SQL_ERROR)
        {
            outError("Prepare error", sStmt);
        }
    }

    for (int i = 0; i < args->test_num; i++)
    {
        if (args->use_prepare)
        {
            prepareExecute(sStmt, args->print_rows);
        }
        else
        {
            directExecute2(sStmt, args->print_rows);
        }
    }

    if (SQL_ERROR == SQLFreeStmt(sStmt, SQL_DROP))
    {
        outError("FreeStmt", sStmt);
    }

    clock_gettime(CLOCK_MONOTONIC, &sEndTime);
    double sElapsedSec = (double)(sEndTime.tv_sec - sStartTime.tv_sec) +
                         (double)(sEndTime.tv_nsec - sStartTime.tv_nsec) / 1000000000.0;
    printf("thread %d elapsed time between connect and disconnect: %.6f sec\n",
           args->thread_index, sElapsedSec);

    db_disconnect();
    return NULL;
}

int main(int argc, char **argv)
{
    char * sHost    = NULL;
    int    sPort    = 0;
    int    sTestNum = 0;
    int    sThreadCount = 0;
    int    sPrintRows = 0;
    int    sUsePrepare = 0;
    int    sArgIdx = 1;

    if (argc >= 2 && strcmp(argv[1], "-p") == 0)
    {
        sUsePrepare = 1;
        sArgIdx++;
    }

    if (argc != (sUsePrepare ? 7 : 6))
    {
        fprintf(stderr, "Usage : ./multi [-p] host port test_num thread_count print_rows\n");
        exit(-1);
    }
    else
    {
        switch (argc - sArgIdx + 1)
        {
            case 6: /* flag + 5 args */
            case 5: /* no flag + 5 args */
                sPrintRows = atoi(argv[sArgIdx + 4]);
            case 4:
                sThreadCount = atoi(argv[sArgIdx + 3]);
            case 3:
                sTestNum = atoi(argv[sArgIdx + 2]);
            case 2:
                sPort = atoi(argv[sArgIdx + 1]);
            case 1:
                sHost = argv[sArgIdx];
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
        args[i].use_prepare = sUsePrepare;

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
