package RestV10

import (
	"time"
	"strconv"
	"encoding/json"

	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gomodule/redigo/redis"
)

var g_psearch_curtime int64
var g_psearch_rkey string

func TR_SearchPasswd(db *sql.DB, rds redis.Conn, reqData map[string]interface{}, resBody map[string]interface{}) int {

	reqBody := reqData["body"].(map[string]interface{})
	
	// check input
	if reqBody["step"] == nil || reqBody["type"] == nil { return 9003 }
	if reqBody["type"].(string) != "phone" && reqBody["type"].(string) != "email" { return 9003 }
	if reqBody["type"].(string) == "phone" && (reqBody["ncode"] == nil || reqBody["phone"] == nil) { return 9003 }
	if reqBody["type"].(string) == "email" && reqBody["email"] == nil { return 9003 }

	// global variable
	g_psearch_curtime = time.Now().UnixNano() / 1000000
	if reqBody["type"].(string) == "phone" {
		g_psearch_rkey = global.Config.Service.Name + ":SearchPasswd:" + common.GetPhoneNumber(reqBody["ncode"].(string), reqBody["phone"].(string))
	} else {
		g_psearch_rkey = global.Config.Service.Name + ":SearchPasswd:" + reqBody["email"].(string)
	}

	var err error
	var rvalue string
	var StepInfo map[string]interface{}

	// Redis에서 캐싱값을 가져온다
	if rvalue, err = redis.String(rds.Do("GET", g_psearch_rkey)); err != nil {
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
		case "1": res_code = _SearchPasswdStep1(db, rds, reqBody, resBody, StepInfo)
		case "2": res_code = _SearchPasswdStep2(db, rds, reqBody, resBody, StepInfo)
		case "3": res_code = _SearchPasswdStep3(db, rds, reqBody, resBody, StepInfo)
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
func _SearchPasswdStep1(db *sql.DB, rds redis.Conn, reqBody map[string]interface{}, resBody map[string]interface{}, StepInfo map[string]interface{}) int {

	// 인증번호 5회 이상 실패인지 확인한다
	if StepInfo != nil && StepInfo["block_time"] != nil {
		blockTime := (int64)(StepInfo["block_time"].(float64))
		if g_psearch_curtime <= blockTime {
			resBody["limit_time"] = blockTime
			return 8005
		}
	}

	var err error
	var stmt *sql.Stmt
	var query, userKey string

	// 이미 존재하는 계정인지 체크한다
	if reqBody["type"].(string) == "phone" {
		query = "SELECT USER_KEY FROM USER_INFO WHERE NCODE = :1 and PHONE = :2 and STATUS <> 'C'";
		if stmt, err = db.Prepare(query); err != nil {
			global.FLog.Println(err)
			return 9901
		}
		defer stmt.Close()

		if err = stmt.QueryRow(reqBody["ncode"].(string), reqBody["phone"].(string)).Scan(&userKey); err != nil {
			if err == sql.ErrNoRows {
				return 8017
			} else {
				global.FLog.Println(err)
				return 9901
			}
		}
		if len(userKey) == 0 { return 8017 }
	} else {
		query = "SELECT count(USER_KEY) FROM USER_INFO WHERE EMAIL = :1 and STATUS <> 'C'";
		if stmt, err = db.Prepare(query); err != nil {
			global.FLog.Println(err)
			return 9901
		}
		defer stmt.Close()

		if err = stmt.QueryRow(reqBody["email"].(string)).Scan(&userKey); err != nil {
			if err == sql.ErrNoRows {
				return 8018
			} else {
				global.FLog.Println(err)
				return 9901
			}
		}
		if len(userKey) == 0 { return 8018 }
	}

	// 인증코드를 생성한다
	code := common.GetCodeNumber(6)

	// 에러카운트가 있으면 가져온다
	var errorCount int
	if StepInfo != nil && StepInfo["errcnt"] != nil {
		errorCount = (int)(StepInfo["errcnt"].(float64))
	}

	if reqBody["type"].(string) == "phone" {

		// Redis에 캐싱값을 기록한다
		mapV := map[string]interface{} {"step": "1", "code": code, "expire": g_psearch_curtime + (int64)(global.SendCodeExpireSecs * 1000), "errcnt": errorCount, "ncode": reqBody["ncode"].(string), "phone": reqBody["phone"].(string), "userkey": userKey}
		jsonStr, _ := json.Marshal(mapV)
		if _, err := rds.Do("SET", g_psearch_rkey, jsonStr); err != nil {
			global.FLog.Println(err)
			return 9901
		}

		// 인증코드를 전송한다
		common.SendCode_Phone(reqBody["ncode"].(string), reqBody["phone"].(string), code)
	} else {

		// Redis에 캐싱값을 기록한다
		mapV := map[string]interface{} {"step": "1", "code": code, "expire": g_psearch_curtime + (int64)(global.SendCodeExpireSecs * 1000), "errcnt": errorCount, "email": reqBody["email"].(string), "userkey": userKey}
		jsonStr, _ := json.Marshal(mapV)
		if _, err := rds.Do("SET", g_psearch_rkey, jsonStr); err != nil {
			global.FLog.Println(err)
			return 9901
		}

		// 인증코드를 전송한다
		common.SendCode_Email(reqBody["email"].(string), code)
	}

	// 응답값을 세팅한다
	resBody["expire"] = global.SendCodeExpireSecs
	resBody["code"] = code

	return 0
}

// ReqData - step: 2
//         - type: Request Type (phone, email)
//         - ncode: type = phone then 국가코드
//         - phone: type = phone then 전화번호
//         - email: type = email then 이메일
//         - code: 6자리 코드
// ResData - ok: true/false
//         - errcnt: 오류횟수
//         - maxerr: 최대 오류횟수
//         - limit_time: 제한시간 (timestamp)
func _SearchPasswdStep2(db *sql.DB, rds redis.Conn, reqBody map[string]interface{}, resBody map[string]interface{}, StepInfo map[string]interface{}) int {
	
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
			if _, err = rds.Do("SET", g_psearch_rkey, jsonStr); err != nil {
				global.FLog.Println(err)
				return 9901
			}

			resBody["ok"] = false
			resBody["errcnt"] = errorCount
			resBody["maxerr"] = global.SendCodeMaxErrors
			return 8008

		} else {		// 오류횟수가 최대허용횟수 이상이라면
			blockTime := g_psearch_curtime + (int64)(global.SendCodeBlockSecs * 1000)
			rvalue = `{"block_time": ` + strconv.FormatInt(blockTime, 10) + `}`
			if _, err = rds.Do("SET", g_psearch_rkey, rvalue); err != nil {
				global.FLog.Println(err)
				return 9901
			}

			resBody["ok"] = false
			resBody["errcnt"] = errorCount
			resBody["maxerr"] = global.SendCodeMaxErrors
			resBody["limit_time"] = blockTime
			return 8005
		}
	}

	// Redis에 캐싱값을 기록한다
	mapV := map[string]interface{} {"step": "2", "userkey": StepInfo["userkey"].(string)}
	jsonStr, _ = json.Marshal(mapV)
	if _, err = rds.Do("SET", g_psearch_rkey, jsonStr); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 응답값을 세팅한다
	resBody["ok"] = true

	return 0
}

// ReqData - step: 3
//         - type: Request Type (phone, email)
//         - ncode: type = phone then 국가코드
//         - phone: type = phone then 전화번호
//         - email: type = email then 이메일
//         - loginpw: 로그인 비밀번호
// ResData - ok: true/false
func _SearchPasswdStep3(db *sql.DB, rds redis.Conn, reqBody map[string]interface{}, resBody map[string]interface{}, StepInfo map[string]interface{}) int {
	
	// check input
	if reqBody["loginpw"] == nil || len(reqBody["loginpw"].(string)) <= 1 { return 9003 }
	
	// check cache
	if StepInfo == nil || StepInfo["step"] == nil || StepInfo["step"].(string) != "2" { return 9902 }
	if StepInfo["userkey"] == nil { return 9902 }

	var err error
	
	// 비밀번호를 변경한다
	_, err = db.Exec("UPDATE USER_INFO SET LOGIN_PASSWD = :1, ERROR_COUNT = 0, LOGIN_LIMIT_TIME = 0, UPDATE_TIME = sysdate WHERE USER_KEY = :2", reqBody["loginpw"].(string), StepInfo["userkey"].(string))
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 캐시 정보는 삭제한다
	if _, err = rds.Do("DEL", g_psearch_rkey); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 응답값을 세팅한다
	resBody["ok"] = true

	return 0
}
