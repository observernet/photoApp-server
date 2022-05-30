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

var g_regist_curtime int64

func TR_RegistEmail(db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	reqBody := reqData["body"].(map[string]interface{})
	
	// check input
	if reqBody["step"] == nil || reqBody["email"] == nil || reqBody["userkey"] == nil { return 9003 }
	if reqBody["proc"] == nil || (reqBody["proc"].(string) != "new" && reqBody["proc"].(string) != "old") { return 9003 }
	if len(reqBody["email"].(string)) == 0 || len(reqBody["userkey"].(string)) == 0 { return 9003 }

	// global variable
	g_regist_curtime = time.Now().UnixNano() / 1000000

	var err error
	var rkey, rvalue string
	var StepInfo map[string]interface{}

	// Redis에서 캐싱값을 가져온다
	rkey = global.Config.Service.Name + ":RegistEmail:" + reqBody["email"].(string)
	if rvalue, err = redis.String(rds.Do("GET", rkey)); err != nil {
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
		case "1": res_code = _RegistEmailStep1(db, rds, reqBody, resBody, StepInfo)
		case "2": res_code = _RegistEmailStep2(db, rds, reqBody, resBody, StepInfo)
		default: res_code = 9003
	}
	
	return res_code
}

// ReqData - step: 1
//         - proc: 처리구분 (new, old)
//         - email: 이메일
//		   - userkey: 회원가입시 전송한 사용자 고유키
// ResData - expire: 만료시간 (초)
//         - limit_time: 제한시간 (timestamp)
//         - code: 인증코드 (6자리) - 임시, 오픈시 삭제할 예정임
func _RegistEmailStep1(db *sql.DB, rds redis.Conn, reqBody map[string]interface{}, resBody map[string]interface{}, StepInfo map[string]interface{}) int {

	// 인증번호 5회 이상 실패인지 확인한다
	if StepInfo != nil && StepInfo["block_time"] != nil {
		blockTime := (int64)(StepInfo["block_time"].(float64))
		if g_regist_curtime <= blockTime {
			resBody["limit_time"] = blockTime
			return 8005
		}
	}

	var err error
	var stmt *sql.Stmt
	var query string
	var userCount int

	// 해당 계정이 존재하는지 체크한다
	query = "SELECT count(USER_KEY) FROM USER_INFO WHERE USER_KEY = :1";
	if stmt, err = db.Prepare(query); err != nil {
		global.FLog.Println(err)
		return 9901
	}
	defer stmt.Close()

	if err = stmt.QueryRow(reqBody["userkey"].(string)).Scan(&userCount); err != nil {
		if err == sql.ErrNoRows {
			userCount = 0
		} else {
			global.FLog.Println(err)
			return 9901
		}
	}
	if userCount == 0 { return 8016 }

	// 이미 존재하는 계정인지 체크한다
	query = "SELECT count(USER_KEY) FROM USER_INFO WHERE EMAIL = :1 and STATUS <> 'C'";
	if stmt, err = db.Prepare(query); err != nil {
		global.FLog.Println(err)
		return 9901
	}
	defer stmt.Close()

	if err = stmt.QueryRow(reqBody["email"].(string)).Scan(&userCount); err != nil {
		if err == sql.ErrNoRows {
			userCount = 0
		} else {
			global.FLog.Println(err)
			return 9901
		}
	}

	if userCount > 0 {
		global.FLog.Println("이미 등록된 이메일입니다 [%s]", reqBody["email"].(string))
		return 8003
	}

	// 인증코드를 생성한다
	code := common.GetCodeNumber(6)

	// 에러카운트가 있으면 가져온다
	var errorCount int
	if StepInfo != nil && StepInfo["errcnt"] != nil {
		errorCount = (int)(StepInfo["errcnt"].(float64))
	}

	// Redis에 캐싱값을 기록한다
	rkey := global.Config.Service.Name + ":RegistEmail:" + reqBody["email"].(string)
	mapV := map[string]interface{} {"step": "1", "code": code, "expire": g_regist_curtime + (int64)(global.SendCodeExpireSecs * 1000), "errcnt": errorCount, "proc": reqBody["proc"].(string), "userkey": reqBody["userkey"].(string)}
	jsonStr, _ := json.Marshal(mapV)
	if _, err = rds.Do("SET", rkey, jsonStr); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 인증코드를 전송한다
	common.SendCode_Email(reqBody["email"].(string), code)

	// 응답값을 세팅한다
	resBody["expire"] = global.SendCodeExpireSecs
	resBody["code"] = code

	return 0
}

// ReqData - step: 2
//         - proc: 처리구분 (new, old)
//         - email: 이메일
//		   - userkey: 회원가입시 전송한 사용자 고유키
//         - code: 핸드폰으로 전송한 6자리 코드
// ResData - ok: true/false
//         - errcnt: 오류횟수
//         - maxerr: 최대 오류횟수
//         - limit_time: 제한시간 (timestamp)
func _RegistEmailStep2(db *sql.DB, rds redis.Conn, reqBody map[string]interface{}, resBody map[string]interface{}, StepInfo map[string]interface{}) int {
	
	// check input
	if reqBody["code"] == nil { return 9003 }

	// check cache
	if StepInfo == nil || StepInfo["step"] == nil || StepInfo["step"].(string) != "1" { return 9902 }
	if StepInfo["code"] == nil || StepInfo["expire"] == nil { return 9902 }
	if StepInfo["proc"].(string) != reqBody["proc"].(string) || StepInfo["userkey"].(string) != reqBody["userkey"].(string) { return 9902 }

	// 타임아웃을 체크
	if g_regist_curtime > (int64)(StepInfo["expire"].(float64)) { return 8007 }

	var err error
	var rkey, rvalue string
	var jsonStr []byte

	// 코드를 체크한다
	rkey = global.Config.Service.Name + ":RegistEmail:" + reqBody["email"].(string)
	if reqBody["code"].(string) != StepInfo["code"].(string) {

		errorCount := (int)(StepInfo["errcnt"].(float64)) + 1

		if errorCount < global.SendCodeMaxErrors {  // 오류횟수가 최대허용횟수 미만이라면
			StepInfo["errcnt"] = errorCount
			jsonStr, _ = json.Marshal(StepInfo)
			if _, err = rds.Do("SET", rkey, jsonStr); err != nil {
				global.FLog.Println(err)
				return 9901
			}

			resBody["ok"] = false
			resBody["errcnt"] = errorCount
			resBody["maxerr"] = global.SendCodeMaxErrors
			return 8008

		} else {		// 오류횟수가 최대허용횟수 이상이라면
			blockTime := g_regist_curtime + (int64)(global.SendCodeBlockSecs * 1000)
			rvalue = `{"block_time": ` + strconv.FormatInt(blockTime, 10) + `}`
			if _, err = rds.Do("SET", rkey, rvalue); err != nil {
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

	// 해당 이메일로 기존 제보앱 정보를 가져온다
	if reqBody["proc"].(string) == "old" {

	}

	// 이메일을 세팅한다
	_, err = db.Exec("UPDATE USER_INFO SET EMAIL = :1, UPDATE_TIME = sysdate WHERE USER_KEY = :2", reqBody["email"].(string), reqBody["userkey"].(string))
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 캐시 정보는 삭제한다
	if _, err = rds.Do("DEL", rkey); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 응답값을 세팅한다
	resBody["ok"] = true

	return 0
}
