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
func TR_WalletOBSR(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	userkey := reqData["key"].(string)
	reqBody := reqData["body"].(map[string]interface{})

	// check input
	if reqBody["loginkey"] == nil { return 9003 }

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

	// 계정 상태 및 로그인 정보를 체크한다
	if mapUser["info"].(map[string]interface{})["STATUS"].(string) != "V" { return 8013 }
	if mapUser["login"].(map[string]interface{})["loginkey"].(string) != reqBody["loginkey"].(string) { return 8014 }

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
	resBody["wallet"] = wallets

	return 0
}
