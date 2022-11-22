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

func TR_UserBlock(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	ctx, cancel := context.WithTimeout(c, global.DBContextTimeout * time.Second)
	defer cancel()

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

	// 본인과 차단사용자가 같은지 체크한다
	if userkey == reqBody["block_userkey"].(string) {
		return 8029
	}

	// 존재하는 USER인지 체크한다
	var count int64 = 0
	err = db.QueryRowContext(ctx, "SELECT count(USER_KEY) FROM USER_INFO WHERE USER_KEY = '" + reqBody["block_userkey"].(string) + "'").Scan(&count)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	if count == 0 {
		return 8030
	}

	// 이미 차단한 USER인지 체크한다
	count = 0
	err = db.QueryRowContext(ctx, "SELECT count(BLOCK_USER_KEY) FROM USER_BLOCK WHERE USER_KEY = '" + userkey + "' and BLOCK_USER_KEY = '" + reqBody["block_userkey"].(string) + "'").Scan(&count)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	if count > 0 {
		return 8031
	}

	// 사용자를 차단한다
	_, err = db.ExecContext(ctx, "INSERT INTO USER_BLOCK (USER_KEY, BLOCK_USER_KEY, BLOCK_TIME) VALUES ('" + userkey + "', '" + reqBody["block_userkey"].(string) + "', sysdate) ")
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	resBody["ok"] = true
	return 0
}
