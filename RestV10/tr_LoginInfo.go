package RestV10

import (
	"time"

	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

// ReqData - loginkey: 로그인키
// ResData - info: 사용자 정보
//         - wallet: 지갑정보
//         - reason: 정책위반사유
func TR_LoginInfo(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	userkey := reqData["key"].(string)
	reqBody := reqData["body"].(map[string]interface{})
	
	// check input
	if reqBody["loginkey"] == nil { return 9003 }
	curtime := time.Now().UnixNano() / 1000000

	var err error

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

	// 계정 상태를 체크한다
	if mapUser["info"].(map[string]interface{})["STATUS"].(string) != "V" {
		if mapUser["info"].(map[string]interface{})["STATUS"].(string) == "A" {
			resBody["reason"] = mapUser["info"].(map[string]interface{})["ABUSE_REASON"].(string)
			return 8012
		} else {
			return 8013
		}
	}

	// 로그인정보가 일치하는지 체크
	if mapUser["login"].(map[string]interface{})["loginkey"].(string) != reqBody["loginkey"].(string) {
		return 8014
	}

	// 관리자변수를 가져온다
	var adminVar global.AdminConfig
	if adminVar, err = common.GetAdminVar(rds); err != nil {
		global.FLog.Println(err)
		return 9901
	}
	remain_snap_time := curtime - (int64)(mapUser["stat"].(map[string]interface{})["LAST_SNAP_TIME"].(float64))
	remain_snap_time = adminVar.Snap.Interval - remain_snap_time / 1000
	if remain_snap_time < 0 { remain_snap_time = 0 }

	// 응답값을 세팅한다
	resBody["info"] = map[string]interface{} {
			"ncode": mapUser["info"].(map[string]interface{})["NCODE"].(string),
			"phone": mapUser["info"].(map[string]interface{})["PHONE"].(string),
			"email": mapUser["info"].(map[string]interface{})["EMAIL"].(string),
			"name":  mapUser["info"].(map[string]interface{})["NAME"].(string),
			"photo": mapUser["info"].(map[string]interface{})["PHOTO"].(string),
			"level": mapUser["info"].(map[string]interface{})["USER_LEVEL"].(float64)}
	
	resBody["stat"] = map[string]interface{} {
		"obsp": mapUser["stat"].(map[string]interface{})["OBSP"].(float64),
		"labels": mapUser["stat"].(map[string]interface{})["LABEL_COUNT"].(float64),
		"remain_snap_time": remain_snap_time,
		"count": map[string]interface{} {
			"snap": mapUser["stat"].(map[string]interface{})["TODAY_SNAP_COUNT"].(float64),
			"snap_rp": mapUser["stat"].(map[string]interface{})["TODAY_SNAP_COUNT"].(float64) * adminVar.Reword.Snap,
			"label": mapUser["stat"].(map[string]interface{})["TODAY_LABEL_COUNT"].(float64),
			"label_rp": mapUser["stat"].(map[string]interface{})["TODAY_LABEL_COUNT"].(float64) * adminVar.Reword.Label,
			"label_etc": mapUser["stat"].(map[string]interface{})["TODAY_LABEL_ETC_COUNT"].(float64),
			"label_etc_rp": mapUser["stat"].(map[string]interface{})["TODAY_LABEL_ETC_COUNT"].(float64) * adminVar.Reword.LabelEtc}}

	wallets := make([]map[string]interface{}, 0)
	if mapUser["wallet"] != nil {
		for _, wallet := range mapUser["wallet"].([]map[string]interface{}) {
			wallets = append(wallets, map[string]interface{} {
											"address": wallet["ADDRESS"].(string),
											"type":    wallet["WALLET_TYPE"].(string),
											"name":    wallet["NAME"].(string)})
		}
	}
	resBody["wallet"] = wallets

	return 0
}
