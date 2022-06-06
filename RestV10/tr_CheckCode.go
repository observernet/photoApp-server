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
//  - code: tr_SendCode에서 보낸 코드
// ResData
//  - ok: true/false
//  - certkey: 인증완료키, 다음단계에서 넣어야 한다
//  - errcnt: 에러횟수
//  - maxerr: 에러제한횟수

func TR_CheckCode(c *gin.Context, db *sql.DB, rds redis.Conn, reqData map[string]interface{}, resBody map[string]interface{}) int {

	reqBody := reqData["body"].(map[string]interface{})
	
	// Check Input
	if reqBody["type"] == nil || (reqBody["type"].(string) != "phone" && reqBody["type"].(string) != "email") { return 9003 }
	if reqBody["type"].(string) == "phone" && (reqBody["ncode"] == nil || reqBody["phone"] == nil) { return 9003 }
	if reqBody["type"].(string) == "email" && reqBody["email"] == nil { return 9003 }
	if reqBody["ncode"] != nil && string(reqBody["ncode"].(string)[0]) == "+" { return 9003 }
	if reqBody["code"] == nil { return 9003 }
	curtime := time.Now().UnixNano() / 1000000

	var err error
	var rkey, rvalue string

	// 전송한 코드 정보를 가져온다
	if reqBody["type"].(string) == "phone" {
		rkey = global.Config.Service.Name + ":SendCode:" + common.GetPhoneNumber(reqBody["ncode"].(string), reqBody["phone"].(string))
	} else {
		rkey = global.Config.Service.Name + ":SendCode:" + reqBody["email"].(string)
	}
	if rvalue, err = redis.String(rds.Do("GET", rkey)); err != nil {
		if err != redis.ErrNil {
			global.FLog.Println(err)
			return 9901
		}
	}
	if len(rvalue) == 0 { return 8006 }

	// 코드 정보를 Map으로 변환한다
	codeMap := make(map[string]interface{})	
	if err = json.Unmarshal([]byte(rvalue), &codeMap); err != nil {
		global.FLog.Println(err)
		return 9901
	}
	
	// 타임아웃을 체크한다
	if curtime > (int64)(codeMap["expire"].(float64)) { return 8007 }

	// 코드를 체크한다
	if reqBody["code"].(string) != codeMap["code"].(string) {

		errorCount := (int)(codeMap["errcnt"].(float64)) + 1

		if errorCount < global.SendCodeMaxErrors {  // 오류횟수가 최대허용횟수 미만이라면
			codeMap["errcnt"] = errorCount
			jsonStr, _ := json.Marshal(codeMap)
			if _, err = rds.Do("SET", rkey, jsonStr); err != nil {
				global.FLog.Println(err)
				return 9901
			}

			resBody["ok"] = false
			resBody["errcnt"] = errorCount
			resBody["maxerr"] = global.SendCodeMaxErrors
			return 8008

		} else {		// 오류횟수가 최대허용횟수 이상이라면

			// 코드정보는 삭제한다
			if _, err = rds.Do("DEL", rkey); err != nil {
				global.FLog.Println(err)
				return 9901
			}

			resBody["ok"] = false
			resBody["errcnt"] = errorCount
			resBody["maxerr"] = global.SendCodeMaxErrors
			return 8009
		}
	}

	// 코드정보는 삭제한다
	if _, err = rds.Do("DEL", rkey); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 응답값을 세팅한다
	resBody["ok"] = true
	if reqBody["type"].(string) == "phone" {
		resBody["certkey"] = common.EncyptData(common.GetPhoneNumber(reqBody["ncode"].(string), reqBody["phone"].(string)), "phone")
	} else {
		resBody["certkey"] = common.EncyptData(reqBody["email"].(string), "email")
	}
	
	return 0
}
