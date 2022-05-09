package RestV10

import (
	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gomodule/redigo/redis"
)

// ReqData - ncode: 국가코드
//         - phone: 핸드폰
// ResData - nodata: 리턴정보 없음
func TR_Logout(db *sql.DB, rds redis.Conn, reqData map[string]interface{}, resBody map[string]interface{}) int {

	key := reqData["key"].(string)
	reqBody := reqData["body"].(map[string]interface{})
	
	// check input
	if reqBody["ncode"] == nil || reqBody["phone"] == nil { return 9003 }
	if reqBody["ncode"] != nil && string(reqBody["ncode"].(string)[0]) == "+" { return 9003 }
	
	var err error

	// 로그인 정보를 가져온다
	var mapUser map[string]interface{}
	if mapUser, err = common.User_GetInfo(rds, key); err != nil {
		if err == redis.ErrNil {
			return 8015
		} else {
			global.FLog.Println(err)
			return 9901
		}
	}

	// 로그인정보가 일치하는지 체크한다
	if mapUser["info"].(map[string]interface{})["NCODE"].(string) != reqBody["ncode"].(string) || mapUser["info"].(map[string]interface{})["PHONE"].(string) != reqBody["phone"].(string) {
		return 8014
	}

	// 로그아웃을 처리한다
	if err = common.User_Logout(rds, key); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 응답값을 세팅한다
	resBody["nodata"] = " "

	return 0
}
