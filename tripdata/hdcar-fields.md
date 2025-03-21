
### CN7 Fields

|    | Field                   |
|----|-------------------------|
|0   |  t[s]                   |
|1   |  WHL_SPD_RR[km/h]       |
|2   |  WHL_SPD_RL[km/h]       |
|3   |  WHL_SPD_FR[km/h]       |
|4   |  WHL_SPD_FL[km/h]       |
|5   |  PRESSURE_RR[PSI]       |
|6   |  PRESSURE_RL[PSI]       |
|7   |  PRESSURE_FR[PSI]       |
|8   |  PRESSURE_FL[PSI]       |
|9   |  SAS_Speed[]            |
|10  |  SAS_Angle[Deg]         |
|11  |  CR_Mdps_StrTq[Nm]      |
|12  |  CR_Mdps_OutTq[]        |
|13  |  YAW_RATE[¢ª/s]         |
|14  |  LONG_ACCEL[m/s^2]      |
|15  |  LAT_ACCEL[m/s^2]       |
|16  |  CF_Clu_VehicleSpeed[]  |
|17  |  CF_Clu_Odometer[km]    |
|18  |  VS[km/h]               |
|19  |  CR_Fatc_OutTemp[¡É]    |
|20  |  RIDEHEIGHT_RR[mm]      |
|21  |  RIDEHEIGHT_RL[mm]      |
|22  |  RIDEHEIGHT_FR[mm]      |
|23  |  RIDEHEIGHT_FL[mm]      |
|24  |  MUL_CODE[]             |
|25  |  DRIVER_FLOOR_FL_AngV_Z[deg/s]   |
|26  |  DRIVER_FLOOR_FL_AngV_Y[deg/s]   |
|27  |  DRIVER_FLOOR_FL_AngV_X[deg/s]   |
|28  |  DRIVER_FLOOR_FL_Acc_Z[m/s^2]    |
|29  |  DRIVER_FLOOR_FL_Acc_Y[m/s^2]    |
|30  |  DRIVER_FLOOR_FL_Acc_X[m/s^2]    |
|31  |  Command[]                    |
|32  |  Longitude_Degrees[Degrees]   |
|33  |  Latitude_Degrees[Degrees]    |
|34  |  DGPS_Active[On]              |
|35  |  Altitude[metres]             |
|36  |  Time[UTC]                    |
|37  |  Sats[Sats]                   |
|38  |  Speed_Kmh[Km/h]              |

### RG3 Fields

|    | Field                        |
|----|------------------------------|
| 1  |   t[s]                       |
| 2  |   TPMS_RRTirePrsrVal[PSI]    |
| 3  |   TPMS_RLTirePrsrVal[PSI]    |
| 4  |   TPMS_FRTirePrsrVal[PSI]    |
| 5  |   TPMS_FLTirePrsrVal[PSI]    |
| 6  |   YRS_YawSigSta[]            |
| 7  |   YRS_YawRtVal[¢ª/s]         |
| 8  |   YRS_SnsrTyp[]              |
| 9  |   YRS_LongAccelVal[g]        |
| 10 |   YRS_LongAccelSigSta[]      |
| 11 |   YRS_LatAccelVal[g]         |
| 12 |   YRS_LatAccelSigSta[]       |
| 13 |   WHL_SpdRRVal[km^h]         |
| 14 |   WHL_SpdRLVal[km^h]         |
| 15 |   WHL_SpdFRVal[km^h]         |
| 16 |   WHL_SpdFLVal[km^h]         |
| 17 |   SAS_SpdVal[Deg/s]          |
| 18 |   SAS_AnglVal[Deg]           |
| 19 |   SAS_AlvCnt1Val[]           |
| 20 |   MDPS_StrTqSnsrVal[Nm]      |
| 21 |   MDPS_PaStrAnglVal[Deg]     |
| 22 |   MDPS_OutTqVal[Nm]          |
| 23 |   MDPS_LoamModSta[]          |
| 24 |   MDPS_EstStrAnglVal[Deg]    |
| 25 |   MDPS_CurrModVal[]          |
| 26 |   CLU_OutTempFSta[]          |
| 27 |   CLU_OutTempCSta[]          |
| 28 |   CLU_OdoVal[km]             |
| 29 |   CLU_DisSpdVal_KPH[km/h]    |
| 30 |   RIDEHEIGHT_RR[mm]          |
| 31 |   RIDEHEIGHT_RL[mm]          |
| 32 |   RIDEHEIGHT_FR[mm]          |
| 33 |   RIDEHEIGHT_FL[mm]          |
| 34 |   MUL_CODE[]                 |
| 35 |   DRIVER_FLOOR_FL_AngV_Z[deg/s]   |
| 36 |   DRIVER_FLOOR_FL_AngV_Y[deg/s]   |
| 37 |   DRIVER_FLOOR_FL_AngV_X[deg/s]   |
| 38 |   DRIVER_FLOOR_FL_Acc_Z[m/s^2]    |
| 39 |   DRIVER_FLOOR_FL_Acc_Y[m/s^2]    |
| 40 |   DRIVER_FLOOR_FL_Acc_X[m/s^2]    |
| 41 |   Command[]                       |
| 42 |   Longitude_Degrees[Degrees]      |
| 43 |   Latitude_Degrees[Degrees]       |
| 44 |   DGPS_Active[On]           |
| 45 |   Altitude[metres]          |
| 46 |   Time[UTC]                 |
| 47 |   Sats[Sats]                |
| 48 |   Speed_Kmh[Km/h]           |


### CAN

|    | Field          |
|----|-----------------------|
| 1  | timestamps    |
| 2  | Warn_AsstStBltSwSta    |
| 3  | Warn_DrvStBltSwSta    |
| 4  | Warn_RrCtrStBltSwSta    |
| 5  | Warn_RrLftStBltSwSta    |
| 6  | Warn_RrRtStBltSwSta    |
| 7  | Wiper_PrkngPosSta    |
| 8  | CLU_DisSpdVal_KPH    |
| 9  | CLU_OdoVal    |
| 10 | DATC_OutTempSnsrVal    |
| 11 | SAS_AnglVal    |
| 12 | WHL_SpdRRVal    |
| 13 | WHL_PlsFLVal    |
| 14 | WHL_PlsFRVal    |
| 15 | WHL_PlsRLVal    |
| 16 | WHL_PlsRRVal    |
| 17 | WHL_DirFLVal    |
| 18 | WHL_DirFRVal    |
| 19 | WHL_DirRLVal    |
| 20 | WHL_DirRRVal    |
| 21 | WHL_SpdFLVal    |
| 22 | WHL_SpdFRVal    |
| 23 | WHL_SpdRLVal    |
| 24 | MCU_Mg1EstTqVal    |
| 25 | MCU_Mg1ActlRotatSpdRpmVal    |
| 26 | IMU_YawRtVal    |
| 27 | IMU_LatAccelVal    |
| 28 | IMU_LongAccelVal    |
| 29 | TPMS_FLTirePrsrVal    |
| 30 | TPMS_FRTirePrsrVal    |
| 31 | TPMS_RLTirePrsrVal    |
| 32 | TPMS_RRTirePrsrVal    |
| 33 | event_dt              |
