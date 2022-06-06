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
func TR_Logout(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	userkey := reqData["key"].(string)
	reqBody := reqData["body"].(map[string]interface{})
	
	// check input
	if reqBody["loginkey"] == nil { return 9003 }
	
	var err error

	// 유저 정보를 가져온다
	//var mapUser map[string]interface{}
	//if mapUser, err = common.User_GetInfo(rds, key); err != nil {
	//	if err == redis.ErrNil {
	//		return 8015
	//	} else {
	//		global.FLog.Println(err)
	//		return 9901
	//	}
	//}

	// 로그인 정보를 가져온다
	var mapLogin map[string]interface{}
	if mapLogin, err = common.User_GetLoginInfo(rds, userkey); err != nil {
		if err == redis.ErrNil {
			return 8015
		} else {
			global.FLog.Println(err)
			return 9901
		}
	}

	// 로그인정보가 일치하는지 체크
	if mapLogin["loginkey"].(string) != reqBody["loginkey"].(string) {
		return 8014
	}

	// 로그아웃을 처리한다
	if err = common.User_Logout(rds, userkey); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 응답값을 세팅한다
	resBody["nodata"] = " "

	return 0
}
