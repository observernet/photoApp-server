package RestV10

import (
	//"encoding/json"
	
	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

// ReqData - 
// ResData - 
func TR_ExchangeInfo(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	userkey := reqData["key"].(string)
	reqBody := reqData["body"].(map[string]interface{})

	// check input
	if reqBody["loginkey"] == nil { return 9003 }

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

	// OBSP 잔액을 가져온다
	obsp, err := common.GetUserOBSP(db, userkey)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 수수료 면제 티켓을 가져온다
	fee_free_ticket := 0

	// 예상 수수료를 계산한다
	mapFee, err := common.GetTxFee(rds, adminVar.TxFee.Exchange.Coin, adminVar.TxFee.Exchange.Fee)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 응답값을 세팅한다
	resBody["obsp"] = obsp
	resBody["txfee"] = mapFee["txfee"].(float64) * 2.0
	resBody["base_txfee"] = adminVar.TxFee.Exchange.Fee
	resBody["obsr_price"] = mapFee["obsr_price"].(float64)
	resBody["obsr_time"] = mapFee["obsr_time"].(float64)
	resBody["klay_price"] = mapFee["klay_price"].(float64)
	resBody["klay_time"] = mapFee["klay_time"].(float64)
	resBody["fee_ticket"] = fee_free_ticket

	return 0
}
