package RestV10

import (
	"time"
	"context"
	"strconv"
	"strings"
	"encoding/json"

	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

var g_search_curtime int64
var g_search_hashkey string

func TR_SearchUser(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	ctx, cancel := context.WithTimeout(c, global.DBContextTimeout * time.Second)
	defer cancel()

	reqBody := reqData["body"].(map[string]interface{})
	
	// check input
	if reqBody["step"] == nil || reqBody["type"] == nil { return 9003 }
	if reqBody["type"].(string) != "phone" && reqBody["type"].(string) != "email" { return 9003 }
	if reqBody["type"].(string) == "phone" && (reqBody["ncode"] == nil || reqBody["phone"] == nil) { return 9003 }
	if reqBody["type"].(string) == "email" && reqBody["email"] == nil { return 9003 }

	// global variable
	g_search_curtime = time.Now().UnixNano() / 1000000
	if reqBody["type"].(string) == "phone" {
		g_search_hashkey = common.GetPhoneNumber(reqBody["ncode"].(string), reqBody["phone"].(string))
	} else {
		g_search_hashkey = reqBody["email"].(string)
	}

	var err error
	var rvalue string
	var StepInfo map[string]interface{}

	// Redis에서 캐싱값을 가져온다
	if rvalue, err = redis.String(rds.Do("HGET", global.Config.Service.Name + ":SendCode:SearchUser", g_search_hashkey)); err != nil {
		if err != redis.ErrNil {
			global.FLog.Println(err)
			return 9901
		}
	}

	// 캐싱데이타가 존재하면 Map 데이타로 변환한다
	if len(rvalue) > 0 {
		StepInfo = make(map[string]interface{})
		if err = json.Unmarshal([]byte(rvalue), &StepInfo); err != nil {
			global.FLog.Println(err)
			return 9901
		}
	}

	var res_code int
	var step string = reqBody["step"].(string)
	switch step {
		case "1": res_code = _SearchUserStep1(ctx, db, rds, reqBody, resBody, StepInfo)
		case "2": res_code = _SearchUserStep2(ctx, db, rds, reqBody, resBody, StepInfo)
		default: res_code = 9003
	}
	
	return res_code
}

// ReqData - step: 1
//         - type: Request Type (phone, email)
//         - ncode: type = phone then 국가코드
//         - phone: type = phone then 전화번호
//         - email: type = email then 이메일
// ResData - expire: 만료시간 (초)
//         - limit_time: 제한시간 (timestamp)
//         - code: 인증코드 (6자리) - 임시, 오픈시 삭제할 예정임
func _SearchUserStep1(ctx context.Context, db *sql.DB, rds redis.Conn, reqBody map[string]interface{}, resBody map[string]interface{}, StepInfo map[string]interface{}) int {

	// 인증번호 5회 이상 실패인지 확인한다
	if StepInfo != nil && StepInfo["block_time"] != nil {
		blockTime := (int64)(StepInfo["block_time"].(float64))
		if g_search_curtime <= blockTime {
			resBody["limit_time"] = blockTime
			return 8005
		}
	}

	// 인증코드를 생성한다
	code := common.GetCodeNumber(6)

	// 에러카운트가 있으면 가져온다
	var errorCount, userCount int
	if StepInfo != nil && StepInfo["errcnt"] != nil {
		errorCount = (int)(StepInfo["errcnt"].(float64))
	}

	if reqBody["type"].(string) == "phone" {

		// 해당 전화번호의 계정이 존재하는지 체크한다
		err := db.QueryRowContext(ctx, "SELECT count(USER_KEY) FROM USER_INFO WHERE NCODE = '" + reqBody["ncode"].(string) + "' and PHONE = '" + reqBody["phone"].(string) + "'").Scan(&userCount)
		if err != nil {
			if err == sql.ErrNoRows {
				userCount = 0
			} else {
				global.FLog.Println(err)
				return 9901
			}
		}
		if userCount == 0 { return 8002 }

		// Redis에 캐싱값을 기록한다
		mapV := map[string]interface{} {"step": "1", "code": code, "expire": g_search_curtime + (int64)(global.SendCodeExpireSecs * 1000), "errcnt": errorCount, "ncode": reqBody["ncode"].(string), "phone": reqBody["phone"].(string)}
		jsonStr, _ := json.Marshal(mapV)
		if _, err := rds.Do("HSET", global.Config.Service.Name + ":SendCode:SearchUser", g_search_hashkey, jsonStr); err != nil {
			global.FLog.Println(err)
			return 9901
		}

		// 인증코드를 전송한다
		if _, err = common.SMSApi_Send(reqBody["ncode"].(string), reqBody["phone"].(string), "SearchUser", code); err != nil {
			global.FLog.Println(err)
			return 9901
		}
	} else {

		// 해당 이메일의 계정이 존재하는지 체크한다
		err := db.QueryRowContext(ctx, "SELECT count(USER_KEY) FROM USER_INFO WHERE EMAIL = '" + reqBody["email"].(string) + "'").Scan(&userCount)
		if err != nil {
			if err == sql.ErrNoRows {
				userCount = 0
			} else {
				global.FLog.Println(err)
				return 9901
			}
		}
		if userCount == 0 { return 8004 }

		// Redis에 캐싱값을 기록한다
		mapV := map[string]interface{} {"step": "1", "code": code, "expire": g_search_curtime + (int64)(global.SendCodeExpireSecs * 1000), "errcnt": errorCount, "email": reqBody["email"].(string)}
		jsonStr, _ := json.Marshal(mapV)
		if _, err := rds.Do("HSET", global.Config.Service.Name + ":SendCode:SearchUser", g_search_hashkey, jsonStr); err != nil {
			global.FLog.Println(err)
			return 9901
		}

		// 인증코드를 전송한다
		if _, err = common.MailApi_SendMail(reqBody["email"].(string), "SearchUser", code); err != nil {
			global.FLog.Println(err)
			return 9901
		}
	}

	// 응답값을 세팅한다
	resBody["expire"] = global.SendCodeExpireSecs
	//resBody["code"] = code

	return 0
}

// ReqData - step: 2
//         - type: Request Type (phone, email)
//         - ncode: type = phone then 국가코드
//         - phone: type = phone then 전화번호
//         - email: type = email then 이메일
//         - code: 6자리 코드
// ResData - ok: true/false
//		   - user: 계정정보
//         - errcnt: 오류횟수
//         - maxerr: 최대 오류횟수
//         - limit_time: 제한시간 (timestamp)
func _SearchUserStep2(ctx context.Context, db *sql.DB, rds redis.Conn, reqBody map[string]interface{}, resBody map[string]interface{}, StepInfo map[string]interface{}) int {
	
	// check input
	if reqBody["code"] == nil { return 9003 }

	// check cache
	if StepInfo == nil || StepInfo["step"] == nil || StepInfo["step"].(string) != "1" { return 9902 }
	if StepInfo["code"] == nil || StepInfo["expire"] == nil { return 9902 }

	// 타임아웃을 체크
	if g_search_curtime > (int64)(StepInfo["expire"].(float64)) { return 8007 }

	var err error
	var rvalue string
	var jsonStr []byte

	// 코드를 체크한다
	if reqBody["code"].(string) != StepInfo["code"].(string) {

		errorCount := (int)(StepInfo["errcnt"].(float64)) + 1

		if errorCount < global.SendCodeMaxErrors {  // 오류횟수가 최대허용횟수 미만이라면
			StepInfo["errcnt"] = errorCount
			jsonStr, _ = json.Marshal(StepInfo)
			if _, err = rds.Do("HSET", global.Config.Service.Name + ":SendCode:SearchUser", g_search_hashkey, jsonStr); err != nil {
				global.FLog.Println(err)
				return 9901
			}

			resBody["errcnt"] = errorCount
			resBody["maxerr"] = global.SendCodeMaxErrors
			return 8008

		} else {		// 오류횟수가 최대허용횟수 이상이라면
			blockTime := g_search_curtime + (int64)(global.SendCodeBlockSecs * 1000)
			rvalue = `{"block_time": ` + strconv.FormatInt(blockTime, 10) + `}`
			if _, err = rds.Do("HSET", global.Config.Service.Name + ":SendCode:SearchUser", g_search_hashkey, rvalue); err != nil {
				global.FLog.Println(err)
				return 9901
			}

			resBody["errcnt"] = errorCount
			resBody["maxerr"] = global.SendCodeMaxErrors
			resBody["limit_time"] = blockTime
			return 8005
		}
	}

	var stmt *sql.Stmt
	var query, userValue string

	if reqBody["type"].(string) == "phone" {
		query = "SELECT EMAIL FROM USER_INFO WHERE NCODE = :1 and PHONE = :2 and STATUS <> 'C'";
		if stmt, err = db.PrepareContext(ctx, query); err != nil {
			global.FLog.Println(err)
			return 9901
		}
		defer stmt.Close()

		if err = stmt.QueryRow(reqBody["ncode"].(string), reqBody["phone"].(string)).Scan(&userValue); err != nil {
			if err == sql.ErrNoRows {
				return 8017
			} else {
				global.FLog.Println(err)
				return 9901
			}
		}

		s := strings.Split(userValue, ".")
		if len(s) == 2 {
			ss := strings.Split(s[0], "@")
			if len(ss) == 2 {
				if len(ss[0]) > 3 {
					userValue = ss[0][0:3]
					for i := 3 ; i < len(ss[0]) ; i++ { userValue = userValue + "*" }
				} else {
					userValue = ss[0]
				}
				userValue = userValue + "@" + ss[1][0:1]
				for i := 1 ; i < len(ss[1]) ; i++ { userValue = userValue + "*" }
				userValue = userValue + "." + s[1]
				//userValue = ss[0][0:3] + "****@" + ss[1][0:1] + "***." + s[1]
			} else {
				userValue = userValue[0:3] + "****";
			}
		} else {
			userValue = userValue[0:3] + "****";
		}
	} else {
		query = "SELECT PHONE FROM USER_INFO WHERE EMAIL = :1 and STATUS <> 'C'";
		if stmt, err = db.PrepareContext(ctx, query); err != nil {
			global.FLog.Println(err)
			return 9901
		}
		defer stmt.Close()

		if err = stmt.QueryRow(reqBody["email"].(string)).Scan(&userValue); err != nil {
			if err == sql.ErrNoRows {
				return 8018
			} else {
				global.FLog.Println(err)
				return 9901
			}
		}

		length := len(userValue)
		if length > 4 { userValue = userValue[length-4:length] }
	}

	// 캐시 정보는 삭제한다
	if _, err = rds.Do("HDEL", global.Config.Service.Name + ":SendCode:SearchUser", g_search_hashkey); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 응답값을 세팅한다
	resBody["user"] = userValue

	return 0
}
