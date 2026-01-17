/******************************************************************************
 * Copyright of this product 2013-2023,
 * MACHBASE Corporation(or Inc.) or its subsidiaries.
 * All Rights reserved.
 ******************************************************************************/

#include <stdio.h>
#include <time.h>
#include <itf.h>
#include <machbase_sqlcli.h>
#define TEST_LOGTABLE "test_logtbl"

SQLHENV gEnv;
SQLHDBC gCon;

int getHashValue(const void *aKey)
{
    int sHigh  = (int)(*(nbp_sint64_t *)aKey >> 32);
    int sLow   = (int)(*(nbp_sint64_t *)aKey & 0xffffffff);
    int sValue = sHigh ^ sLow;

    sValue += ~(sValue << 15);
    sValue ^=  (sValue >> 10);
    sValue +=  (sValue << 3);
    sValue ^=  (sValue >> 6);
    sValue += ~(sValue << 11);
    sValue ^=  (sValue >> 16);

    return sValue;
}

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
        //SQLFreeConnect(gCon);
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


void ExecuteSQL(const char *aSQL, int aErrIgnore)
{
    SQLHSTMT sStmt;

    if (SQLAllocStmt(gCon, &sStmt) == SQL_ERROR)
    {
        if (aErrIgnore != 0) return;
        outError("AllocStmt", sStmt);
    }

    if (SQLExecDirect(sStmt, (SQLCHAR *)aSQL, SQL_NTS) == SQL_ERROR)
    {
        if (aErrIgnore != 0) return;
        printf("sql_exec_direct error[%s] \n", aSQL);
        outError("sql_exec_direct error", sStmt);
    }

    if (SQL_ERROR == SQLFreeStmt(sStmt, SQL_DROP))
    {
        if (aErrIgnore != 0) return;
        outError("FreeStmt", sStmt);
    }
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

int directExecute()
{
    const char *sSQL = SQL_STR;

    SQLHSTMT    sStmt = SQL_NULL_HSTMT;

    SQLLEN      sIdLen      = 0;
    SQLLEN      sValueLen   = 0;
    SQLLEN      sRegDateLen = 0;

    char                 sId[33];
    double               sValue;
    SQL_TIMESTAMP_STRUCT sRegDate;

    if( SQLAllocStmt(gCon, &sStmt) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, sStmt, "SQLAllocStmt Error");
        goto error;
    }

    if( SQLPrepare(sStmt, (SQLCHAR *)sSQL, SQL_NTS) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, sStmt, "SQPrepare Error");
        goto error;
    }

    if( SQLExecute(sStmt) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, sStmt, "SQLExecute Error");
        goto error;
    }

    if( SQLBindCol(sStmt, 1, SQL_C_CHAR, sId, sizeof(sId), &sIdLen) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, sStmt, "SQLBindCol 1 Error");
        goto error;
    }

    if( SQLBindCol(sStmt, 2, SQL_C_TYPE_TIMESTAMP, &sRegDate, 0, &sRegDateLen) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, sStmt, "SQLBindCol 2 Error");
        goto error;
    }

    if( SQLBindCol(sStmt, 3, SQL_C_DOUBLE, &sValue, 0, &sValueLen) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, sStmt, "SQLBindCol 3 Error");
        goto error;
    }

    /* printf("--------------------------------------------------------------------------------\n"); */
    /* printf("%-33s%-33s%-33s\n","NAME","TIME","VALUE"); */
    /* printf("--------------------------------------------------------------------------------\n"); */

    while( SQLFetch(sStmt) == SQL_SUCCESS )
    {
        /* printf("%-33s", sId); */
        /* printf("%d-%02d-%02d %02d:%02d:%02d 000:000:000 ", */
        /*             sRegDate.year, sRegDate.month, sRegDate.day, */
        /*             sRegDate.hour, sRegDate.minute, sRegDate.second); */
        /* printf("%-.2lf", sValue); */

        /* printf("\n"); */
    }

    if( SQLFreeStmt(sStmt, SQL_DROP) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, sStmt, "SQLFreeStmt Error");
        goto error;
    }
    sStmt = SQL_NULL_HSTMT;

    return 0;

error:
    if( sStmt != SQL_NULL_HSTMT )
    {
        SQLFreeStmt(sStmt, SQL_DROP);
        sStmt = SQL_NULL_HSTMT;
    }

    return -1;
}



int prepareExecute(SQLHSTMT *aStmt)
{
    const char *sSQL = SQL_STR;

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

    /* printf("--------------------------------------------------------------------------------\n"); */
    /* printf("%-33s%-33s%-33s\n","NAME","TIME","VALUE"); */
    /* printf("--------------------------------------------------------------------------------\n"); */

    while( SQLFetch(aStmt) == SQL_SUCCESS )
    {
        /* printf("%-33s", sId); */
        /* printf("%d-%02d-%02d %02d:%02d:%02d 000:000:000 ", */
        /*             sRegDate.year, sRegDate.month, sRegDate.day, */
        /*             sRegDate.hour, sRegDate.minute, sRegDate.second); */
        /* printf("%-.2lf", sValue); */

        /* printf("\n"); */
    }

    /* if( SQLFreeStmt(aStmt, SQL_DROP) != SQL_SUCCESS ) */
    /* { */
    /*     printError(gEnv, gCon, aStmt, "SQLFreeStmt Error"); */
    /*     goto error; */
    /* } */
    /* aStmt = SQL_NULL_HSTMT; */

    return 0;

error:
    if( aStmt != SQL_NULL_HSTMT )
    {
        SQLFreeStmt(aStmt, SQL_DROP);
        aStmt = SQL_NULL_HSTMT;
    }

    return -1;
}

int fetch(SQLHSTMT *aStmt)
{
    SQLLEN      sIdLen      = 0;
    SQLLEN      sValueLen   = 0;
    SQLLEN      sRegDateLen = 0;

    char                 sId[33];
    double               sValue;
    SQL_TIMESTAMP_STRUCT sRegDate;

    printf("--------------------------------------------------------------------------------\n");
    printf("%-33s%-33s%-33s\n","NAME","TIME","VALUE");
    printf("--------------------------------------------------------------------------------\n");

    while( SQLFetch(aStmt) == SQL_SUCCESS )
    {
        printf("%-33s", sId);
        printf("%d-%02d-%02d %02d:%02d:%02d 000:000:000 ",
                    sRegDate.year, sRegDate.month, sRegDate.day,
                    sRegDate.hour, sRegDate.minute, sRegDate.second);
        printf("%-.2lf", sValue);

        printf("\n");
    }

    /* if( SQLFreeStmt(sStmt, SQL_DROP) != SQL_SUCCESS ) */
    /* { */
    /*     printError(gEnv, gCon, sStmt, "SQLFreeStmt Error"); */
    /*     goto error; */
    /* } */
    /* sStmt = SQL_NULL_HSTMT; */

    return 0;

error:
    if( aStmt != SQL_NULL_HSTMT )
    {
        SQLFreeStmt(aStmt, SQL_DROP);
    }

    return -1;
}


int main(int argc, char **argv)
{
    char * sHost    = NULL;
    int    sPort    = 0;
    int    sTestNum = 0;
    int    sRows    = 0;
    struct timespec sStartTime;
    struct timespec sEndTime;

    if (argc != 4)
    {
        fprintf(stderr, "Usage : ./append host port rows test_num\n");
        exit(-1);
    }
    else
    {
        switch (argc)
        {
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

    db_connect(sHost, sPort);
    clock_gettime(CLOCK_MONOTONIC, &sStartTime);

    SQLHSTMT sStmt;

    if (SQLAllocStmt(gCon, &sStmt) == SQL_ERROR)
    {
        outError("AllocStmt", sStmt);
    }

    if (SQLPrepare(sStmt, SQL_STR, SQL_NTS) == SQL_ERROR)
    {
        outError("Prepare error", sStmt);
    }

    /* fprintf(stderr, "wait..>"); */
    /* getc(stdin); */

    for (int i = 0; i <sTestNum; i++)
    {
        directExecute();

        //prepareExecute(sStmt);
    }

    if (SQL_ERROR == SQLFreeStmt(sStmt, SQL_DROP))
    {
        outError("FreeStmt", sStmt);
    }


    clock_gettime(CLOCK_MONOTONIC, &sEndTime);
    double sElapsedSec = (double)(sEndTime.tv_sec - sStartTime.tv_sec) +
                         (double)(sEndTime.tv_nsec - sStartTime.tv_nsec) / 1000000000.0;
    printf("elapsed time between connect and disconnect: %.6f sec\n", sElapsedSec);

    db_disconnect();

    return 0;
}
