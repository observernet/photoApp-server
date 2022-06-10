package RestV10

import (
	//"encoding/json"
	
	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

// ReqData - loginkey: 로그인키
//         - snapkey: 스냅키
//         - status: 사진상태 (0:정상, 1:신고)
//         - sky: 하늘상태 (N:없음, Y:있음, F:가득)
//         - rain: 강수상태 (N:없음, Y:비, S:눈)
//         - wcondi: 자연현상 (N:없음, F:안개, H:우박, R:무지개, T:번개)
//         - calamity: 재난현상 (N:없음, R:폭우, S:폭설, F:산불, L:산사태)
//         - accuse: 신고구분 (V:날씨확인불가, F:직접촬영한날씨사진아님, A:광고, S:음란폭력혐오, P:초상권침해)
//         - is_etc: 기타라벨링여부 (true, false)
// ResData - snapkey: 스냅키
//         - labelkey: 라벨키
//         - rp: 오늘 RP {snap: }
func TR_Label(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	userkey := reqData["key"].(string)
	reqBody := reqData["body"].(map[string]interface{})

	// check input
	if reqBody["loginkey"] == nil || reqBody["snapkey"] == nil { return 9003 }
	if reqBody["status"] == nil || (reqBody["status"].(string) != "0" && reqBody["status"].(string) != "1") { return 9003 }
	if reqBody["status"].(string) == "0" && (reqBody["sky"] == nil || reqBody["rain"] == nil) { return 9003 }
	if reqBody["status"].(string) == "1" && reqBody["accuse"] == nil { return 9003 }
	if reqBody["is_etc"] != nil && reqBody["is_etc"].(bool) == true && (reqBody["wcondi"] == nil || reqBody["calamity"] == nil) { return 9003 }

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

	// 관리자 정의변수를 가져온다
	//var adminVar global.AdminConfig
	//if adminVar, err = common.GetAdminVar(rds); err != nil {
	//	global.FLog.Println(err)
	//	return 9901
	//}

	// 계정 상태 및 로그인 정보를 체크한다
	if mapUser["info"].(map[string]interface{})["STATUS"].(string) != "V" { return 8013 }
	if mapUser["login"].(map[string]interface{})["loginkey"].(string) != reqBody["loginkey"].(string) { return 8014 }



	return 0
}
