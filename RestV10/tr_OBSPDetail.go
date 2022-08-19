package RestV10

import (
	"strconv"
	//"encoding/json"
	
	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

type _OBSPDetail_APPData struct {
	T struct {
		Reword			float64
		RPTotal			float64
		CountUser		int64
	}
	U struct {
		Reword			float64
		RPTotal			float64
		RPSnap			float64
		RPLabel			float64
		RPLabelEtc		float64
		CountSnap		int64
		CountLabel		int64
		CountLabelEtc	int64
	}
}

type _OBSPDetail_WSData struct {
	SerialNo			string
	Reword				float64
	Address				string
	Time				int64
}

type _OBSPDetail_ExData struct {
	Amount				float64
	Fee					float64
}

// ReqData - 
// ResData - 
func TR_OBSPDetail(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	userkey := reqData["key"].(string)
	reqBody := reqData["body"].(map[string]interface{})

	// check input
	if reqBody["loginkey"] == nil || reqBody["date"] == nil { return 9003 }


	var err error

	// 관리자 정의변수를 가져온다
	var adminVar global.AdminConfig
	if adminVar, err = common.GetAdminVar(rds); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 유저 정보를 가져온다
	var mapUser map[string]interface{}
	if mapUser, err = common.User_GetInfo(rds, userkey); err != nil {
		if err == redis.ErrNil {
			return 8015
		} else {
			global.FLog.Println(err)
			return 9901
		}
	}

	// 계정 상태 및 로그인 정보를 체크한다
	if mapUser["info"].(map[string]interface{})["STATUS"].(string) != "V" { return 8013 }
	if mapUser["login"].(map[string]interface{})["loginkey"].(string) != reqBody["loginkey"].(string) { return 8014 }

	// 조회일의 앱 전체 보상정보를 가져온다
	var reword_idx int64
	var APP _OBSPDetail_APPData
	query := "SELECT REWORD_IDX, TOTAL_REWORD, TOTAL_RP, COUNT_USER FROM REWORD_LIST WHERE DEVICE = 'P' AND REWORD_DATE = " + reqBody["date"].(string) + " AND PROC_STATUS = 'V'";
	err = db.QueryRow(query).Scan(&reword_idx, &APP.T.Reword, &APP.T.RPTotal, &APP.T.CountUser)
	if err != nil && err != sql.ErrNoRows {
		global.FLog.Println(err)
		return 9901
	}

	// 조회일의 앱 사용자 보상정보를 가져온다
	if reword_idx > 0 {
		query := "SELECT REWORD_AMOUNT, TOTAL_RP, SNAP_RP, LABEL_RP, LABEL_ETC_RP, COUNT_SNAP, COUNT_LABEL, COUNT_LABEL_ETC " +
				"FROM REWORD_DETAIL " +
				"WHERE REWORD_IDX = " + strconv.FormatInt(reword_idx, 10) + " " +
				"  AND USER_KEY = '" + userkey + "'";
		err = db.QueryRow(query).Scan(&APP.U.Reword, &APP.U.RPTotal, &APP.U.RPSnap, &APP.U.RPLabel, &APP.U.RPLabelEtc, &APP.U.CountSnap, &APP.U.CountLabel, &APP.U.CountLabelEtc)
		if err != nil && err != sql.ErrNoRows {
			global.FLog.Println(err)
			return 9901
		}
	}
	
	// APP 정보를 세팅한다
	if APP.T.RPTotal > 0 {
		resBody["APP"] = map[string]interface{} {
							"obsp": APP.U.Reword,
							"stat": map[string]interface{} {
									"my_rp": APP.U.RPTotal,
									"total_rp": APP.T.RPTotal,
									"total_reword": APP.T.Reword,
									"my_action": (APP.U.RPTotal / APP.T.RPTotal) * 100,
									"avg_action": ((APP.T.RPTotal / (float64)(APP.T.CountUser)) / APP.T.RPTotal) * 100},
							"snap": map[string]interface{} {"rp": APP.U.RPSnap, "cnt": APP.U.CountSnap},
							"label": map[string]interface{} {"rp": APP.U.RPLabel, "cnt": APP.U.CountLabel, "cnt_pass": APP.U.RPLabel / adminVar.Reword.Label},
							"label_etc": map[string]interface{} {"rp": APP.U.RPLabelEtc, "cnt": APP.U.CountLabelEtc, "cnt_pass": APP.U.RPLabelEtc / adminVar.Reword.LabelEtc}}
	}

	// 조회일의 WS 보상정보를 가져온다
	var WS _OBSPDetail_WSData
	query = "SELECT B.SERIAL_NO, B.REWORD_AMOUNT FROM REWORD_LIST A, REWORD_DETAIL B WHERE A.REWORD_IDX = B.REWORD_IDX and DEVICE = 'W' and REWORD_DATE = " + reqBody["date"].(string) + " and PROC_STATUS = 'V'";
	rows, err := db.Query(query)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	defer rows.Close()

	var WSSerialList string
	WSList := make([]_OBSPDetail_WSData, 0)
	for rows.Next() {	
		err = rows.Scan(&WS.SerialNo, &WS.Reword)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}

		WSSerialList = WSSerialList + WS.SerialNo + ","
		WSList = append(WSList, WS)
	}

	// WS의 주소정보를 가져와 세팅한다
	if len(WSSerialList) > 0 {
		WSSerialList = WSSerialList[0:len(WSSerialList)-1]
		WSAPiRes, err := common.WSApi_GetWSData(WSSerialList)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}
		//global.FLog.Println(WSAPiRes)

		for _, mapWS := range WSAPiRes["body"].([]interface{}) {
			for i := 0 ; i < len(WSList) ; i++ {
				if WSList[i].SerialNo == mapWS.(map[string]interface {})["serial"].(string) {
					WSList[i].Address = mapWS.(map[string]interface {})["address"].(string)
					WSList[i].Time = (int64)(mapWS.(map[string]interface {})["time"].(float64)) * 1000
					break
				}
			}
		}
	}

	// WS 정보를 세팅한다
	if len(WSList) > 0 {
		resWS := make([]map[string]interface{}, 0)
		for i := 0 ; i < len(WSList) ; i++ {
			resWS = append(resWS, map[string]interface{} {"serial": WSList[i].SerialNo, "address": WSList[i].Address, "time": WSList[i].Time, "obsp": WSList[i].Reword})
		}
		resBody["WS"] = resWS
	}

	// 조회일의 환전 정보를 가져온다
	var ExData _OBSPDetail_ExData
	query = "SELECT NVL(SUM(PROC_AMOUNT), 0), NVL(SUM(EXCHANGE_FEE), 0) FROM EXCHANGE_OBSP WHERE USER_KEY = '" + userkey + "' and TO_CHAR(REQ_TIME, 'RRRRMMDD') = " + reqBody["date"].(string) + " and PROC_STATUS = 'V'";
	err = db.QueryRow(query).Scan(&ExData.Amount, &ExData.Fee)
	if err != nil && err != sql.ErrNoRows {
		global.FLog.Println(err)
		return 9901
	}

	// 환전 정보를 세팅한
	resBody["exchange"] = map[string]interface{} {"amount": ExData.Amount, "fee": ExData.Fee}

	return 0
}
