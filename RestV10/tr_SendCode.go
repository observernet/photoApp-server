package RestV10

import (
	"time"
	"encoding/json"

	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

// ReqData
//  - type: Request Type (phone, email)
//  - ncode: type = phone then 국가코드
//  - phone: type = phone then 전화번호
//  - email: type = email then 이메일
// ResData
//  - expire: 만료시간 (초)
//  - code: 인증코드 (6자리) - 임시, 오픈시 삭제할 예정임

func TR_SendCode(c *gin.Context, db *sql.DB, rds redis.Conn, reqData map[string]interface{}, resBody map[string]interface{}) int {

	reqBody := reqData["body"].(map[string]interface{})
	
	// Check Input
	if reqBody["type"] == nil || (reqBody["type"].(string) != "phone" && reqBody["type"].(string) != "email") { return 9003 }
	if reqBody["type"].(string) == "phone" && (reqBody["ncode"] == nil || reqBody["phone"] == nil) { return 9003 }
	if reqBody["type"].(string) == "email" && reqBody["email"] == nil { return 9003 }
	if reqBody["ncode"] != nil && string(reqBody["ncode"].(string)[0]) == "+" { return 9003 }
	curtime := time.Now().UnixNano() / 1000000

	// 인증코드를 생성한다
	code := common.GetCodeNumber(6)

	// 코드를 전송한다
	if reqBody["type"].(string) == "phone" {
		//common.SendCode_Phone(reqBody["ncode"].(string), reqBody["phone"].(string), code)
	} else {
		//common.SendCode_Email(reqBody["email"].(string), code)
	}

	var err error
	var rkey string

	// 정보를 기록할 Redis키를 결정한다
	if reqBody["type"].(string) == "phone" {
		rkey = global.Config.Service.Name + ":SendCode:" + common.GetPhoneNumber(reqBody["ncode"].(string), reqBody["phone"].(string))
	} else {
		rkey = global.Config.Service.Name + ":SendCode:" + reqBody["email"].(string)
	}

	// 코드 정보를 기록한다
	mapV := map[string]interface{} {"code": code, "expire": curtime + (int64)(global.SendCodeExpireSecs * 1000), "errcnt": 0}
	jsonStr, _ := json.Marshal(mapV)
	if _, err = rds.Do("SET", rkey, jsonStr); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 응답값을 세팅한다
	resBody["expire"] = global.SendCodeExpireSecs
	resBody["code"] = code
	
	return 0
}
