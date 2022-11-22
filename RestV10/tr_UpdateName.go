package RestV10

import (
	"time"
	"context"
	"strings"

	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

// ReqData - 
// ResData - 
func TR_UpdateName(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	ctx, cancel := context.WithTimeout(c, global.DBContextTimeout * time.Second)
	defer cancel()

	userkey := reqData["key"].(string)
	reqBody := reqData["body"].(map[string]interface{})
	
	// check input
	if reqBody["loginkey"] == nil { return 9003 }
	if reqBody["name"] == nil || len(reqBody["name"].(string)) == 0 { return 9003 }
	
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

	// 이전값과 동일한지 체크한다
	if strings.EqualFold(reqBody["name"].(string), mapUser["info"].(map[string]interface{})["NAME"].(string)) {
		return 8027
	}

	// 닉네임이 이미 존재하는지 체크한다
	var count int64
	err = db.QueryRowContext(ctx, "SELECT count(USER_KEY) FROM USER_INFO WHERE UPPER(NAME) = '" + strings.ToUpper(reqBody["name"].(string)) + "' and STATUS <> 'C'").Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			count = 0
		} else {
			global.FLog.Println(err)
			return 9901
		}
	}
	if count > 0 { return 8022 }

	// 닉네임에 금칙어가 있는지 체크한다
	pass, err := common.CheckForbiddenWord(ctx, db, "N", reqBody["name"].(string))
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	if ( pass == true ) { return 8023 }

	// 이름 정보를 갱신한다
	query := "UPDATE USER_INFO SET NAME = '" + reqBody["name"].(string) + "', UPDATE_TIME = sysdate WHERE USER_KEY = '" + userkey + "'"
	_, err = db.ExecContext(ctx, query)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// REDIS 사용자 정보를 갱신한다
	if err = common.User_UpdateInfo(ctx, db, rds, userkey); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 응답값을 세팅한다
	resBody["ok"] = true
	
	return 0
}
