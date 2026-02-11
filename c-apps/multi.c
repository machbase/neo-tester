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

int    gPrintRows = 0;


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
    if (gPrintRows != 0)
    {
        printf("connected ... \n");
    }
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

//#define SQL_STR  "select * from tag where name = 'TAG_00' and time between to_date('2017-01-01 00:00:00 000:000:000', 'YYYY-MM-DD HH24:MI:SS mmm:uuu:nnn') and to_date('2017-01-02 00:00:00 000:000:000','YYYY-MM-DD HH24:MI:SS mmm:uuu:nnn')"
#define SQL_STR  "select * from tag where name = 'TAG_00' and time between '2017-01-01 00:00:00 000:000:000' and '2017-01-02 00:00:00 000:000:000'"
#define PREPARE_SQL_STR "select * from tag where name = ? and time between ? and ? limit ?"
#define PREPARE_TAG_NAME "TAG_00"
#define PREPARE_START_TIME "2017-01-01 00:00:00 000:000:000"
#define PREPARE_END_TIME "2017-01-02 00:00:00 000:000:000"
#define PREPARE_LIMIT_COUNT 100
//#define SQL_STR  "select * from v$version limit 1"
//#define SQL_STR  "commit"

struct fetch_buffer {
    SQLLEN      sIdLen;
    SQLLEN      sValueLen;
    SQLLEN      sRegDateLen;
    char        sId[33];
    double      sValue;
    SQL_TIMESTAMP_STRUCT sRegDate;
};

struct prepare_bind_param {
    SQLLEN      sTagNameLen;
    SQLLEN      sStartTimeLen;
    SQLLEN      sEndTimeLen;
    SQLLEN      sLimitCountLen;
    char        sTagName[33];
    char        sStartTime[64];
    char        sEndTime[64];
    SQLINTEGER  sLimitCount;
};

int bindPrepareParameters(SQLHSTMT aStmt, struct prepare_bind_param *aParam)
{
    if (SQLBindParameter(aStmt, 1, SQL_PARAM_INPUT,
                         SQL_C_CHAR, SQL_VARCHAR,
                         strlen(aParam->sTagName), 0,
                         aParam->sTagName, sizeof(aParam->sTagName), &aParam->sTagNameLen) != SQL_SUCCESS)
    {
        printError(gEnv, gCon, aStmt, "SQLBindParameter 1 Error");
        goto error;
    }

    if (SQLBindParameter(aStmt, 2, SQL_PARAM_INPUT,
                         SQL_C_CHAR, SQL_VARCHAR,
                         strlen(aParam->sStartTime), 0,
                         aParam->sStartTime, sizeof(aParam->sStartTime), &aParam->sStartTimeLen) != SQL_SUCCESS)
    {
        printError(gEnv, gCon, aStmt, "SQLBindParameter 2 Error");
        goto error;
    }

    if (SQLBindParameter(aStmt, 3, SQL_PARAM_INPUT,
                         SQL_C_CHAR, SQL_VARCHAR,
                         strlen(aParam->sEndTime), 0,
                         aParam->sEndTime, sizeof(aParam->sEndTime), &aParam->sEndTimeLen) != SQL_SUCCESS)
    {
        printError(gEnv, gCon, aStmt, "SQLBindParameter 3 Error");
        goto error;
    }

    if (SQLBindParameter(aStmt, 4, SQL_PARAM_INPUT,
                         SQL_C_LONG, SQL_INTEGER,
                         0, 0,
                         &aParam->sLimitCount, 0, &aParam->sLimitCountLen) != SQL_SUCCESS)
    {
        printError(gEnv, gCon, aStmt, "SQLBindParameter 4 Error");
        goto error;
    }

    return 0;

error:
    return -1;
}

int bindFetchColumns(SQLHSTMT aStmt, struct fetch_buffer *aFetch)
{
    if( SQLBindCol(aStmt, 1, SQL_C_CHAR, aFetch->sId, sizeof(aFetch->sId), &aFetch->sIdLen) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, aStmt, "SQLBindCol 1 Error");
        goto error;
    }

    if( SQLBindCol(aStmt, 2, SQL_C_TYPE_TIMESTAMP, &aFetch->sRegDate, 0, &aFetch->sRegDateLen) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, aStmt, "SQLBindCol 2 Error");
        goto error;
    }

    if( SQLBindCol(aStmt, 3, SQL_C_DOUBLE, &aFetch->sValue, 0, &aFetch->sValueLen) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, aStmt, "SQLBindCol 3 Error");
        goto error;
    }

    return 0;

error:
    return -1;
}

int directExecute2(SQLHSTMT aStmt, int aPrint, struct fetch_buffer *aFetch)
{
    const char *sSQL = SQL_STR;

    if( SQLExecDirect(aStmt, (SQLCHAR *)sSQL, SQL_NTS) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, aStmt, "SQLExecDirect Error");
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
            printf("%-33s", aFetch->sId);
            printf("%d-%02d-%02d %02d:%02d:%02d 000:000:000 ",
                        aFetch->sRegDate.year, aFetch->sRegDate.month, aFetch->sRegDate.day,
                        aFetch->sRegDate.hour, aFetch->sRegDate.minute, aFetch->sRegDate.second);
            printf("%-.2lf", aFetch->sValue);

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

int prepareExecute(SQLHSTMT aStmt, int aPrint, struct fetch_buffer *aFetch)
{
    if( SQLExecute(aStmt) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, aStmt, "SQLExecute Error");
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
            printf("%-33s", aFetch->sId);
            printf("%d-%02d-%02d %02d:%02d:%02d 000:000:000 ",
                        aFetch->sRegDate.year, aFetch->sRegDate.month, aFetch->sRegDate.day,
                        aFetch->sRegDate.hour, aFetch->sRegDate.minute, aFetch->sRegDate.second);
            printf("%-.2lf", aFetch->sValue);

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
    int prepare_mode; /* 0: direct, 1: -p, 2: -p2 */
};

void *run_thread(void *arg)
{
    struct thread_args *args = (struct thread_args *)arg;
    struct timespec sStartTime;
    struct timespec sEndTime;
    SQLHSTMT sStmt;
    struct fetch_buffer sFetch;
    struct prepare_bind_param sPrepareParam;

    memset(&sFetch, 0, sizeof(sFetch));
    memset(&sPrepareParam, 0, sizeof(sPrepareParam));

    db_connect(args->host, args->port);
    clock_gettime(CLOCK_MONOTONIC, &sStartTime);

    if (SQLAllocStmt(gCon, &sStmt) == SQL_ERROR)
    {
        outError("AllocStmt", sStmt);
    }

    snprintf(sPrepareParam.sTagName, sizeof(sPrepareParam.sTagName), "%s", PREPARE_TAG_NAME);
    snprintf(sPrepareParam.sStartTime, sizeof(sPrepareParam.sStartTime), "%s", PREPARE_START_TIME);
    snprintf(sPrepareParam.sEndTime, sizeof(sPrepareParam.sEndTime), "%s", PREPARE_END_TIME);
    sPrepareParam.sTagNameLen = SQL_NTS;
    sPrepareParam.sStartTimeLen = SQL_NTS;
    sPrepareParam.sEndTimeLen = SQL_NTS;
    sPrepareParam.sLimitCount = PREPARE_LIMIT_COUNT;
    sPrepareParam.sLimitCountLen = 0;

    if (args->prepare_mode == 1)
    {
        if (SQLPrepare(sStmt, (SQLCHAR *)SQL_STR, SQL_NTS) == SQL_ERROR)
        {
            outError("Prepare error", sStmt);
        }
    }

    if (args->prepare_mode != 2 && bindFetchColumns(sStmt, &sFetch) != 0)
    {
        outError("BindCol", sStmt);
    }

    for (int i = 0; i < args->test_num; i++)
    {
        if (args->prepare_mode == 1)
        {
            prepareExecute(sStmt, args->print_rows, &sFetch);
        }
        else if (args->prepare_mode == 2)
        {
            if (SQLPrepare(sStmt, (SQLCHAR *)PREPARE_SQL_STR, SQL_NTS) == SQL_ERROR)
            {
                outError("Prepare error", sStmt);
            }

            if (bindPrepareParameters(sStmt, &sPrepareParam) != 0)
            {
                outError("BindParameter", sStmt);
            }

            if (bindFetchColumns(sStmt, &sFetch) != 0)
            {
                outError("BindCol", sStmt);
            }

            prepareExecute(sStmt, args->print_rows, &sFetch);

            /* if (SQLFreeStmt(sStmt, SQL_CLOSE) == SQL_ERROR) */
            /* { */
            /*     outError("FreeStmt(SQL_CLOSE)", sStmt); */
            /* } */
        }
        else
        {
            directExecute2(sStmt, args->print_rows, &sFetch);
        }
    }

    /* if (SQL_ERROR == SQLFreeStmt(sStmt, SQL_DROP)) */
    /* { */
    /*     outError("FreeStmt", sStmt); */
    /* } */

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
    int    sPrepareMode = 0;
    int    sArgIdx = 1;

    if (argc >= 2 && strcmp(argv[1], "-p") == 0)
    {
        sPrepareMode = 1;
        sArgIdx++;
    }
    else if (argc >= 2 && strcmp(argv[1], "-p2") == 0)
    {
        sPrepareMode = 2;
        sArgIdx++;
    }

    if (argc != (sPrepareMode ? 7 : 6))
    {
        fprintf(stderr, "Usage : ./multi [-p|-p2] host port test_num thread_count print_rows\n");
        exit(-1);
    }
    else
    {
        switch (argc - sArgIdx + 1)
        {
            case 6: /* flag + 5 args */
            case 5: /* no flag + 5 args */
                gPrintRows = atoi(argv[sArgIdx + 4]);
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

    struct timespec start_ts;
    struct timespec end_ts;

    clock_gettime(CLOCK_MONOTONIC, &start_ts);

    for (int i = 0; i < sThreadCount; i++)
    {
        args[i].host = sHost;
        args[i].port = sPort;
        args[i].test_num = sTestNum;
        args[i].thread_index = i;
        args[i].print_rows = gPrintRows;
        args[i].prepare_mode = sPrepareMode;

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

    clock_gettime(CLOCK_MONOTONIC, &end_ts);

    double total_elapsed = (double)(end_ts.tv_sec - start_ts.tv_sec) +
                           (double)(end_ts.tv_nsec - start_ts.tv_nsec) / 1000000000.0;
    long long total_requests = (long long)sTestNum * (long long)sThreadCount;
    double rps = total_elapsed > 0.0 ? (double)total_requests / total_elapsed : 0.0;

    printf("multi total elapsed time: %.6f sec\n", total_elapsed);
    printf("multi total throughput: %.2f req/sec (%lld requests)\n", rps, total_requests);

    free(threads);
    free(args);

    return 0;
}
