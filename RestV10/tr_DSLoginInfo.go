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

// ReqData - loginkey: 로그인키
// ResData - info: 사용자 정보
//         - wallet: 지갑정보
//         - reason: 정책위반사유
func TR_DSLoginInfo(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	ctx, cancel := context.WithTimeout(c, global.DBContextTimeout * time.Second)
	defer cancel()

	// Check Header
	if reqData["comm"] == nil || reqData["comm"].(string) != global.Comm_DataStore {
		return 9005
	}
	userkey := reqData["key"].(string)
	//reqBody := reqData["body"].(map[string]interface{})
	
	// check input
	//if reqBody["loginkey"] == nil { return 9003 }
	//curtime := time.Now().UnixNano() / 1000000

	var err error

	// 로그인 정보를 가져온다
	//var mapLogin map[string]interface{}
	if _, err = common.DSUser_GetLoginInfo(rds, userkey); err != nil {
		if err == redis.ErrNil {
			return 8015
		} else {
			global.FLog.Println(err)
			return 9901
		}
	}

	// 로그인정보가 일치하는지 체크
	//if mapLogin["loginkey"].(string) != reqBody["loginkey"].(string) {
	//	return 8014
	//}

	// 사용자 정보를 가져온다
	var mapUser map[string]interface{}
	if mapUser, err = common.DSUser_GetInfo(ctx, db, rds, userkey); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 응답값을 세팅한다
	resBody["userkey"] = mapUser["userkey"].(string)
	resBody["info"] = mapUser["info"].(map[string]interface{})
	resBody["wallet"] = mapUser["wallet"].([]map[string]interface{})

	return 0
}
