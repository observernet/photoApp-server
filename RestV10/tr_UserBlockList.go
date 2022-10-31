package RestV10

import (
	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

func TR_UserBlockList(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	userkey := reqData["key"].(string)
	reqBody := reqData["body"].(map[string]interface{})

	// check input
	if reqBody["loginkey"] == nil { return 9003 }

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

	// 차단 내역을 가져온다
	rows, err := db.Query("SELECT A.BLOCK_USER_KEY, B.NAME FROM USER_BLOCK A, USER_INFO B WHERE A.BLOCK_USER_KEY = B.USER_KEY and A.USER_KEY = '" + userkey + "'")
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	defer rows.Close()

	var block_userkey, name string

	list := make([]map[string]interface{}, 0)
	for rows.Next() {	
		err = rows.Scan(&block_userkey, &name)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}
		list = append(list, map[string]interface{} {
					"userkey": block_userkey,
					"name": name,
					"photo": "https://photoapp.obsr-app.org/Image/View/profile/" + block_userkey})
	}
	resBody["list"] = list

	return 0
}
