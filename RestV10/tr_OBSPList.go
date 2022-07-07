package RestV10

import (
	//"encoding/json"
	
	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

// ReqData - 
// ResData - 
func TR_OBSPList(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	userkey := reqData["key"].(string)
	reqBody := reqData["body"].(map[string]interface{})

	// check input
	if reqBody["loginkey"] == nil { return 9003 }


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

	// 첫조회라면, OBSP정보를 가져온다
	var mapOBSP map[string]interface{}
	if reqBody["next"] == nil || len(reqBody["next"].(string)) == 0 {

		var adminVar global.AdminConfig

		// 관리자 정의변수를 가져온다
		if adminVar, err = common.GetAdminVar(rds); err != nil {
			global.FLog.Println(err)
			return 9901
		}

		// OBSP 잔액을 가져온다
		obsp, err := common.GetUserOBSP(db, userkey)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}

		mapOBSP = map[string]interface{} {"curr": obsp, "auto": adminVar.Reword.AutoExchange}
	}

	// 보상/환전 내역을 가져온다



	// 응답값을 세팅한다
	if mapOBSP != nil { resBody["obsp"] = mapOBSP }

	return 0
}
