package RestV10

import (
	"time"
	"context"
	"strings"
	
	"photoApp-server/global"
	//"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

// ReqData - 
// ResData - 
func TR_FcmUpdate(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	ctx, cancel := context.WithTimeout(c, global.DBContextTimeout * time.Second)
	defer cancel()

	userkey := reqData["key"].(string)
	reqBody := reqData["body"].(map[string]interface{})

	// check input
	if reqBody["token"] == nil { return 9003 }
	if reqBody["type"] == nil || (reqBody["type"].(string) != "ios" && reqBody["type"].(string) != "android") { return 9003 }

	var err error

	// 토큰정보를 갱신한다
	if len(userkey) == 16 {
		_, err = db.ExecContext(ctx, "DELETE FROM FCM_TOKEN WHERE APP_NAME = 'PhotoApp' and USER_KEY = :1", userkey)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}

		_, err = db.ExecContext(ctx, "INSERT INTO FCM_TOKEN (APP_NAME, TOKEN, TYPE, USER_KEY, TOKEN_ERR, UPDATE_TIME) VALUES ('PhotoApp', :1, :2, :3, 'N', sysdate)",
						 reqBody["token"].(string), reqBody["type"].(string), userkey)
		if err != nil {
			if strings.Contains(err.Error(), "ORA-00001") {
				_, err = db.ExecContext(ctx, "UPDATE FCM_TOKEN SET TYPE = :1, USER_KEY = :2, TOKEN_ERR = 'N', UPDATE_TIME = sysdate WHERE APP_NAME = 'PhotoApp' and TOKEN = :3",
								 reqBody["type"].(string), userkey, reqBody["token"].(string))
				if err != nil {
					global.FLog.Println(err)
					return 9901
				}
			} else {
				global.FLog.Println(err)
				return 9901
			}
		}
	} else {
		_, err = db.ExecContext(ctx, "INSERT INTO FCM_TOKEN (APP_NAME, TOKEN, TYPE, USER_KEY, TOKEN_ERR, UPDATE_TIME) VALUES ('PhotoApp', :1, :2, :3, 'N', sysdate)",
						 reqBody["token"].(string), reqBody["type"].(string), "")
		if err != nil {
			if strings.Contains(err.Error(), "ORA-00001") {
				_, err = db.ExecContext(ctx, "UPDATE FCM_TOKEN SET TYPE = :1, USER_KEY = '', TOKEN_ERR = 'N', UPDATE_TIME = sysdate WHERE APP_NAME = 'PhotoApp' and TOKEN = :2",
								 reqBody["type"].(string), reqBody["token"].(string))
				if err != nil {
					global.FLog.Println(err)
					return 9901
				}
			} else {
				global.FLog.Println(err)
				return 9901
			}
		}
	}
	
	
	// 응답값을 세팅한다
	resBody["ok"] = true

	return 0
}
