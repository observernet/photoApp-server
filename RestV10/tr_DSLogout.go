package RestV10

import (
	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

// ReqData - loginkey: 로그인키
// ResData - nodata: 리턴정보 없음
func TR_DSLogout(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	// Check Header
	if reqData["comm"] == nil || reqData["comm"].(string) != global.Comm_DataStore {
		return 9005
	}
	userkey := reqData["key"].(string)
	//reqBody := reqData["body"].(map[string]interface{})
	
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

	// 로그아웃을 처리한다
	if err = common.DSUser_Logout(rds, userkey); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 응답값을 세팅한다
	resBody["nodata"] = " "

	return 0
}
