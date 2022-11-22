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
// ResData - android: 안드로이드버전
//         - ios: IOS버전
//		   - web: 웹버전
func TR_CheckName(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	ctx, cancel := context.WithTimeout(c, global.DBContextTimeout * time.Second)
	defer cancel()

	reqBody := reqData["body"].(map[string]interface{})
	
	// check input
	if reqBody["name"] == nil || len(reqBody["name"].(string)) <= 1 { return 9003 }

	var err error

	// 닉네임이 이미 존재하는지 체크한다
	var count int64
	err = db.QueryRowContext(ctx, "SELECT count(USER_KEY) FROM USER_INFO WHERE NAME = '" + strings.ToUpper(reqBody["name"].(string)) + "'").Scan(&count)
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

	resBody["ok"] = true
	return 0
}
