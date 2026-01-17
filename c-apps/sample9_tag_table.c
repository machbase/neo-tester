#include <stdio.h>
#include <stdlib.h>
#include <stdarg.h>
#include <string.h>
#include <arpa/inet.h>
#include <sys/time.h>
#include <machbase_sqlcli.h>

#define ERROR_CHECK_COUNT       100000

#define RC_SUCCESS              0
#define RC_FAILURE              -1

#define UNUSED(aVar) do { (void)(aVar); } while(0)

#define CHECK_APPEND_RESULT(aRC, aEnv, aCon, aSTMT)             \
    if( !SQL_SUCCEEDED(aRC) )                                   \
    {                                                           \
        if( checkAppendError(aEnv, aCon, aSTMT) == RC_FAILURE ) \
        {                                                       \
            ;                                                   \
        }                                                       \
    }

typedef struct tm timestruct;

SQLHENV     gEnv;
SQLHDBC     gCon;
SQLHDBC     gLotDataConn;

static char          gTargetIP[16] = "127.0.0.1";
static int           gPortNo=5656;
static unsigned long gMaxData=1000000;
static long          gTps=3000000;
static unsigned int  gEquipCnt = 1;
static unsigned int  gTagPerEq = 10;
static int           gDataPerSec = 1;
int                  gNoLotNo=0;

time_t getTimeStamp();
void printError(SQLHENV aEnv, SQLHDBC aCon, SQLHSTMT aStmt, char *aMsg);
int connectDB();
void disconnectDB();
int executeDirectSQL(const char *aSQL, int aErrIgnore);
int createTable();
int appendOpen(SQLHSTMT aStmt);
int appendData(SQLHSTMT aStmt);
void appendTps(SQLHSTMT aStmt);
unsigned long appendClose(SQLHSTMT aStmt);
int selectTable();

time_t getTimeStamp()
{
    struct timeval sTimeVal;
    int            sRet;

    sRet = gettimeofday(&sTimeVal, NULL);

    if (sRet == 0)
    {
        return (time_t)(sTimeVal.tv_sec * 1000000 + sTimeVal.tv_usec);
    }
    else
    {
        return 0;
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

int checkAppendError(SQLHENV aEnv, SQLHDBC aCon, SQLHSTMT aStmt)
{
    SQLINTEGER      sNativeError;
    SQLCHAR         sErrorMsg[SQL_MAX_MESSAGE_LENGTH + 1];
    SQLCHAR         sSqlState[SQL_SQLSTATE_SIZE + 1];
    SQLSMALLINT     sMsgLength;

    if( SQLError(aEnv, aCon, aStmt, sSqlState, &sNativeError,
        sErrorMsg, SQL_MAX_MESSAGE_LENGTH, &sMsgLength) != SQL_SUCCESS )
    {
        return RC_FAILURE;
    }

    printf("SQLSTATE-[%s], Machbase-[%d][%s]\n", sSqlState, sNativeError, sErrorMsg);

    if( sNativeError != 9604 &&
        sNativeError != 9605 &&
        sNativeError != 9606 )
    {
        return RC_FAILURE;
    }

    return RC_SUCCESS;
}

void appendDumpError(SQLHSTMT    aStmt,
                 SQLINTEGER  aErrorCode,
                 SQLPOINTER  aErrorMessage,
                 SQLLEN      aErrorBufLen,
                 SQLPOINTER  aRowBuf,
                 SQLLEN      aRowBufLen)
{
    char       sErrMsg[1024] = {0, };
    char       sRowMsg[32 * 1024] = {0, };

    UNUSED(aStmt);

    if (aErrorMessage != NULL)
    {
        strncpy(sErrMsg, (char *)aErrorMessage, aErrorBufLen);
    }  

    if (aRowBuf != NULL)
    {
        strncpy(sRowMsg, (char *)aRowBuf, aRowBufLen);
    }

    fprintf(stdout, "Append Error : [%d][%s]\n[%s]\n\n", aErrorCode, sErrMsg, sRowMsg);
}

int connectDB()
{
    char sConnStr[1024];

    if( SQLAllocEnv(&gEnv) != SQL_SUCCESS ) 
    {
        printf("SQLAllocEnv error\n");
        return RC_FAILURE;
    }

    if( SQLAllocConnect(gEnv, &gCon) != SQL_SUCCESS ) 
    {
        printf("SQLAllocConnect error\n");

        SQLFreeEnv(gEnv);
        gEnv = SQL_NULL_HENV;

        return RC_FAILURE;
    }

    sprintf(sConnStr,"SERVER=%s;UID=SYS;PWD=MANAGER;CONNTYPE=1;PORT_NO=%d", gTargetIP, gPortNo);

    if( SQLDriverConnect( gCon, NULL,
                          (SQLCHAR *)sConnStr,
                          SQL_NTS,
                          NULL, 0, NULL,
                          SQL_DRIVER_NOPROMPT ) != SQL_SUCCESS
      )
    {

        printError(gEnv, gCon, NULL, "SQLDriverConnect error");

        SQLFreeConnect(gCon);
        gCon = SQL_NULL_HDBC;

        SQLFreeEnv(gEnv);
        gEnv = SQL_NULL_HENV;

        return RC_FAILURE;
    }

    return RC_SUCCESS;
}

void disconnectDB()
{
    if( SQLDisconnect(gCon) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, NULL, "SQLDisconnect error");
    }

    SQLFreeConnect(gCon);
    gCon = SQL_NULL_HDBC;

    SQLFreeEnv(gEnv);
    gEnv = SQL_NULL_HENV;
}

int executeDirectSQL(const char *aSQL, int aErrIgnore)
{
    SQLHSTMT sStmt = SQL_NULL_HSTMT;

    if( SQLAllocStmt(gCon, &sStmt) != SQL_SUCCESS )
    {
        if( aErrIgnore == 0 )
        {
            printError(gEnv, gCon, sStmt, "SQLAllocStmt Error");
            return RC_FAILURE;
        }
    }
    
    if( SQLExecDirect(sStmt, (SQLCHAR *)aSQL, SQL_NTS) != SQL_SUCCESS )
    {

        if( aErrIgnore == 0 )
        {
            printError(gEnv, gCon, sStmt, "SQLExecDirect Error");

            SQLFreeStmt(sStmt,SQL_DROP);
            sStmt = SQL_NULL_HSTMT;
            return RC_FAILURE;
        }
    }

    if( SQLFreeStmt(sStmt, SQL_DROP) != SQL_SUCCESS )
    {
        if (aErrIgnore == 0)
        {
            printError(gEnv, gCon, sStmt, "SQLFreeStmt Error");
            sStmt = SQL_NULL_HSTMT;
            return RC_FAILURE;
        }
    }

    sStmt = SQL_NULL_HSTMT;
    return RC_SUCCESS;
}

int createTable()
{
    int sRC;

    sRC = executeDirectSQL("DROP TABLE TAG", 1);
    if( sRC != RC_SUCCESS )
    {
        sRC = 0;
    }

    sRC = executeDirectSQL("CREATE TAG TABLE TAG(NAME VARCHAR(32) PRIMARY KEY, TIME DATETIME BASETIME, VALUE DOUBLE SUMMARIZED)", 0);
    if( sRC != RC_SUCCESS )
    {
        return RC_FAILURE;
    }

    return RC_SUCCESS;
}

int appendOpen(SQLHSTMT aStmt)
{
    const char *sTableName = "TAG";

    if( SQLAppendOpen(aStmt, (SQLCHAR *)sTableName, ERROR_CHECK_COUNT) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, aStmt, "SQLAppendOpen Error");
        return RC_FAILURE;
    }

    return RC_SUCCESS;
}

int appendData(SQLHSTMT aStmt)
{
    SQL_APPEND_PARAM *sParam;
    SQLRETURN        sRC;
    unsigned long    i;
    unsigned int     j,p;
    unsigned long    sRecCount = 0;
    char             sTagName[20];
    int              sTag;
    double           sValue;

    int               year,month,hour,min,sec,day;

    struct tm         sTm;
    unsigned long     sTime;
    int               sInterval;
    long              StartTime;
    int               sBreak = 0;

    year     =    2019;
    month    =    0;
    day      =    1;
    hour     =    0;
    min      =    0;
    sec      =    0;

    memset(&sTm, 0, sizeof(struct tm));
    sTm.tm_year = year - 1900;
    sTm.tm_mon  = month;
    sTm.tm_mday = day;
    sTm.tm_hour = hour;
    sTm.tm_min  = min;
    sTm.tm_sec  = sec;

    sTime = mktime(&sTm);
    sTime = sTime * MACHBASE_UINT64_LITERAL(1000000000); 

    if (gNoLotNo == 0)
    {
        sParam = malloc(sizeof(SQL_APPEND_PARAM) * 4);
        memset(sParam, 0, sizeof(SQL_APPEND_PARAM) *4);
    }
    else
    {
        sParam = malloc(sizeof(SQL_APPEND_PARAM) * 3);
        memset(sParam, 0, sizeof(SQL_APPEND_PARAM)*3);
    }
    sInterval = (int)(1000000000/gDataPerSec); // 100ms default.
    
    StartTime = getTimeStamp();
    for( i = 0; (gMaxData == 0) || sBreak == 0; i++ )
    {
        unsigned int sEq = 0;
    
        for( j=0; j< (gEquipCnt * gTagPerEq); j++)
        {
            /* tag_id */
            sTag = j;
            sTagName[0]=0;
            snprintf(sTagName, 20, "EQ%d^TAG%d",sEq, sTag);
            sParam[0].mVar.mLength   = strnlen(sTagName,20);
            sParam[0].mVar.mData     = sTagName;
            sParam[1].mDateTime.mTime =  sTime;
    
            p = j%20;
            /* value */
            switch(p)
            {
                case 1:
                case 6:
                    sValue = ((rand()%501)*0.01)+20; //20 ~ 25;
                    break;
                case 2:
                case 7:
                    sValue = ((rand()%901)*0.01)+30; //30 ~ 39;
                    break;
                case 3:
                case 8:
                    sValue = ((rand()%1501)*0.01)+50; //50 ~ 65;
                    break;
                case 4:
                case 9:
                    sValue = ((rand()%1501)*0.01)+1000;
                    break;
                case 11:
                case 16:
                    sValue = 31.2;
                    break;
                case 12:
                case 17:
                    sValue = 234.567;
                    break;
                default:
                    sValue = (rand()%20000)/100.0; //0 ~ 200
                    break;
            }
            sParam[2].mDouble = sValue;
            sRC = SQLAppendDataV2(aStmt, sParam);
            sRecCount++;
            CHECK_APPEND_RESULT(sRC, gEnv, gCon, aStmt);
            if ((gTps != 0) && (sRecCount % 10 == 0))
            {
                long usecperev = 1000000/gTps;
                long sleepus;
                long elapsed = getTimeStamp() - StartTime;
                sleepus = (usecperev * i) - elapsed;
                if (sleepus > 0)
                {
                    struct timespec sleept;
                    struct timespec leftt;
                    sleept.tv_sec = 0;
                    sleept.tv_nsec = sleepus * 1000;
                    nanosleep(&sleept, &leftt);
                }
            }
    
            if (sTag % gTagPerEq == 0 && sTag != 0)
            {
                sEq ++;
                if (sEq == gEquipCnt) sEq = 0;
                
            }
            
            if (sRecCount > gMaxData - 1)
            {
                goto exit;
            }
        }
        sTime = sTime + sInterval;
    }

exit:
    return RC_SUCCESS;
}

unsigned long appendClose(SQLHSTMT aStmt)
{
    SQLBIGINT sSuccessCount = 0;
    SQLBIGINT sFailureCount = 0;

    if( SQLAppendClose(aStmt, (SQLBIGINT *)&sSuccessCount, (SQLBIGINT *)&sFailureCount) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, aStmt, "SQLAppendClose Error");
        return RC_FAILURE;
    }
    else
    {
        printf("success : %ld, failure : %ld\n", sSuccessCount, sFailureCount);

        return sSuccessCount;
    }
}

int selectTable()
{
    const char *sSQL = "SELECT * FROM TAG WHERE NAME IN ('EQ0^TAG2', 'EQ0^TAG9') AND TIME BETWEEN TO_DATE('2019-01-01 00:00:00') AND TO_DATE('2019-01-01 00:00:09')";

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

    printf("--------------------------------------------------------------------------------\n");
    printf("%-33s%-33s%-33s\n","NAME","TIME","VALUE");
    printf("--------------------------------------------------------------------------------\n");
    
    while( SQLFetch(sStmt) == SQL_SUCCESS )
    {
        printf("%-33s", sId);
        printf("%d-%02d-%02d %02d:%02d:%02d 000:000:000 ",
                    sRegDate.year, sRegDate.month, sRegDate.day,
                    sRegDate.hour, sRegDate.minute, sRegDate.second);
        printf("%-.2lf", sValue);
        
        printf("\n");
    }

    if( SQLFreeStmt(sStmt, SQL_DROP) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, sStmt, "SQLFreeStmt Error");
        goto error;
    }
    sStmt = SQL_NULL_HSTMT;

    return RC_SUCCESS;

error:
    if( sStmt != SQL_NULL_HSTMT )
    {
        SQLFreeStmt(sStmt, SQL_DROP);
        sStmt = SQL_NULL_HSTMT;
    }

    return RC_FAILURE;
}

int main()
{
    SQLHSTMT    sStmt = SQL_NULL_HSTMT;

    unsigned long  sCount=0;
    time_t         sStartTime, sEndTime;

    if( connectDB() == RC_SUCCESS )
    {
        printf("connectDB success\n");
    }
    else
    {
        printf("connectDB failure\n");
        goto error;
    }
    
    if( createTable() == RC_SUCCESS )
    {
        printf("createTable success\n");
    }
    else
    {
        printf("createTable failure\n");
        goto error;
    }


    if( SQLAllocStmt(gCon, &sStmt) != SQL_SUCCESS ) 
    {
        printError(gEnv, gCon, sStmt, "SQLAllocStmt Error");
        goto error;
    }

    if( appendOpen(sStmt) == RC_SUCCESS )
    {
        printf("appendOpen success\n");
    }
    else
    {
        printf("appendOpen failure\n");
        goto error;
    }

    if( SQLAppendSetErrorCallback(sStmt, appendDumpError) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, sStmt, "SQLAppendSetErrorCallback Error");
        goto error;
    }

    sStartTime = getTimeStamp();
    if( appendData(sStmt) != SQL_SUCCESS )
    {
        printf("append failure");
        goto error;
    }
    sEndTime = getTimeStamp();

    sCount = appendClose(sStmt);
    if( sCount > 0 )
    {
        printf("appendClose success\n");
        printf("timegap = %ld microseconds for %ld records\n", sEndTime - sStartTime, sCount);
        printf("%.2f records/second\n",  ((double)sCount/(double)(sEndTime - sStartTime))*1000000);
    }
    else
    {
        printf("appendClose failure\n");
    }

    if( SQLFreeStmt(sStmt, SQL_DROP) != SQL_SUCCESS )
    {
        printError(gEnv, gCon, sStmt, "SQLFreeStmt Error");
        goto error;
    }
    sStmt = SQL_NULL_HSTMT;

    if ( selectTable() != RC_SUCCESS )
    {
        printf("selectTable failure\n");
        goto error;
    }

    disconnectDB();

    return RC_SUCCESS;

error:
    if( sStmt != SQL_NULL_HSTMT )
    {
        SQLFreeStmt(sStmt, SQL_DROP);
        sStmt = SQL_NULL_HSTMT;
    }

    if( gCon != SQL_NULL_HDBC )
    {
        disconnectDB();
    }

    return RC_FAILURE;
}
