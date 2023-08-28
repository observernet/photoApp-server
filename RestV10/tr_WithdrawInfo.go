package RestV10

import (
	"time"
	"context"
	
	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

// ReqData - 
// ResData - 
func TR_WithdrawInfo(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	ctx, cancel := context.WithTimeout(c, global.DBContextTimeout * time.Second)
	defer cancel()

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

	// 수수료 면제 티켓을 가져온다
	var fee_free_ticket int64
	err = db.QueryRowContext(ctx, "SELECT NVL(WTICKET_COUNT, 0) FROM USER_INFO WHERE USER_KEY = '" + userkey + "'").Scan(&fee_free_ticket)
	if err != nil {
		if err == sql.ErrNoRows {
			fee_free_ticket = 0
		} else {
			global.FLog.Println(err)
			return 9901
		}
	}

	// 예상 수수료를 계산한다
	mapFee, err := common.GetTxFee(rds, adminVar.TxFee.Withdraw.Coin, adminVar.TxFee.Withdraw.Fee)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 지갑 정보를 가져온다
	wallets := make([]map[string]interface{}, 0)
	if mapUser["wallet"] != nil {
		for _, wallet := range mapUser["wallet"].([]map[string]interface{}) {

			balance, err := common.KAS_GetBalanceOf(userkey, wallet["ADDRESS"].(string), wallet["WALLET_TYPE"].(string))
			if err != nil {
				global.FLog.Println(err)
				return 9901
			}

			wallets = append(wallets, map[string]interface{} {
				"address": wallet["ADDRESS"].(string),
				"obsr": common.GetFloat64FromString(balance)})
		}
	}

	// 응답값을 세팅한다
	resBody["wallet"] = wallets
	resBody["txfee"] = common.RoundFloat64(mapFee["txfee"].(float64), global.OBSR_PDesz)
	resBody["base_txfee"] = adminVar.TxFee.Withdraw.Fee
	resBody["obsr_price"] = common.RoundFloat64(mapFee["obsr_price"].(float64), global.OBSR_PDesz)
	resBody["obsr_time"] = mapFee["obsr_time"].(float64)
	resBody["fee_ticket"] = fee_free_ticket
	
	return 0
}
