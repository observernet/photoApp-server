package RestV10

import (
	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

func TR_UserBlockClear(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	userkey := reqData["key"].(string)
	reqBody := reqData["body"].(map[string]interface{})

	// check input
	if reqBody["loginkey"] == nil { return 9003 }
	if reqBody["block_userkey"] == nil || len(reqBody["block_userkey"].(string)) != 16 { return 9003 }

	// 유저 정보를 가져온다
	mapUser, err := common.User_GetInfo(rds, userkey, "info", "login")
	if err != nil {
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

	// 이미 차단한 USER인지 체크한다
	var count int64 = 0
	err = db.QueryRow("SELECT count(BLOCK_USER_KEY) FROM USER_BLOCK WHERE USER_KEY = '" + userkey + "' and BLOCK_USER_KEY = '" + reqBody["block_userkey"].(string) + "'").Scan(&count)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	if count == 0 {
		return 8032
	}

	// 사용자를 차단을 해제한다
	_, err = db.Exec("DELETE FROM USER_BLOCK WHERE USER_KEY = '" + userkey + "' and BLOCK_USER_KEY = '" + reqBody["block_userkey"].(string) + "'")
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	resBody["ok"] = true
	return 0
}
