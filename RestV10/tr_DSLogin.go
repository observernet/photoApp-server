package RestV10

import (
	"time"
	"context"
	"strconv"
	"encoding/json"

	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

var g_DSlogin_curtime int64
var g_DSlogin_hashkey string

func TR_DSLogin(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	ctx, cancel := context.WithTimeout(c, global.DBContextTimeout * time.Second)
	defer cancel()

	reqBody := reqData["body"].(map[string]interface{})
	
	// check input
	if reqBody["step"] == nil { return 9003 }
	if reqBody["type"] == nil || (reqBody["type"].(string) != "phone" && reqBody["type"].(string) != "email") { return 9003 }
	if reqBody["type"].(string) == "phone" && (reqBody["ncode"] == nil || reqBody["phone"] == nil) { return 9003 }
	if reqBody["type"].(string) == "email" && reqBody["email"] == nil { return 9003 }
	if reqBody["type"].(string) == "phone" && reqBody["ncode"] != nil && string(reqBody["ncode"].(string)[0]) == "+" { return 9003 }

	// Global 변수값을 세팅한다
	g_login_curtime = time.Now().UnixNano() / 1000000
	if reqBody["type"].(string) == "phone" {
		g_login_hashkey = common.GetPhoneNumber(reqBody["ncode"].(string), reqBody["phone"].(string))
	} else {
		g_login_hashkey = reqBody["email"].(string)
	}

	var err error
	var rvalue string
	var loginInfo map[string]interface{}

	// Redis에서 캐싱값을 가져온다
	if rvalue, err = redis.String(rds.Do("HGET", "DataStore:SendCode:Login", g_login_hashkey)); err != nil {
		if err != redis.ErrNil {
			global.FLog.Println(err)
			return 9901
		}
	}

	// 캐싱데이타가 존재하면 Map 데이타로 변환한다
	if len(rvalue) > 0 {
		loginInfo = make(map[string]interface{})
		if err = json.Unmarshal([]byte(rvalue), &loginInfo); err != nil {
			global.FLog.Println(err)
			return 9901
		}
	}

	var res_code int
	var step string = reqBody["step"].(string)
	switch step {
		case "1": res_code = _DSLoginStep1(ctx, db, rds, reqBody, resBody, loginInfo)
		case "2": res_code = _DSLoginStep2(ctx, db, rds, reqBody, resBody, loginInfo)
		default: res_code = 9003
	}
	
	return res_code
}

//
// ReqData - step: 1
//         - type: Request Type (phone, email)
//         - ncode: type == phone, 국가코드
//         - phone: type == phone, 핸드폰
//         - email: type == email, 이메일
// ResData - expire: 만료시간 (초)
//         - limit_time: 제한시간 (timestamp)
//         - reason: 정책위반사유
//         - code: 인증코드 (6자리) - 임시, 오픈시 삭제할 예정임
func _DSLoginStep1(ctx context.Context, db *sql.DB, rds redis.Conn, reqBody map[string]interface{}, resBody map[string]interface{}, loginInfo map[string]interface{}) int {

	// 인증번호 5회 이상 실패인지 확인한다
	if loginInfo != nil && loginInfo["block_time"] != nil {
		blockTime := (int64)(loginInfo["block_time"].(float64))
		if g_login_curtime <= blockTime {
			resBody["limit_time"] = blockTime
			return 8005
		}
	}

	var err error
	var stmt *sql.Stmt
	var query string
	var userkey, status, reason string

	// 계정정보를 가져온다
	if reqBody["type"].(string) == "phone" {
		query = "SELECT USER_KEY, STATUS, ABUSE_REASON FROM USER_INFO WHERE NCODE = :1 and PHONE = :2 and STATUS <> 'C'";
		if stmt, err = db.PrepareContext(ctx, query); err != nil {
			global.FLog.Println(err)
			return 9901
		}
		defer stmt.Close()

		if err = stmt.QueryRow(reqBody["ncode"].(string), reqBody["phone"].(string)).Scan(&userkey, &status, &reason); err != nil {
			if err == sql.ErrNoRows {
				return 8010
			} else {
				global.FLog.Println(err)
				return 9901
			}
		}
	} else {
		query = "SELECT USER_KEY, STATUS, ABUSE_REASON FROM USER_INFO WHERE EMAIL = :1 and STATUS <> 'C'";
		if stmt, err = db.PrepareContext(ctx, query); err != nil {
			global.FLog.Println(err)
			return 9901
		}
		defer stmt.Close()

		if err = stmt.QueryRow(reqBody["email"].(string)).Scan(&userkey, &status, &reason); err != nil {
			if err == sql.ErrNoRows {
				return 8010
			} else {
				global.FLog.Println(err)
				return 9901
			}
		}
	}

	// 계정 상태에 따라
	if status != "V" {
		if status == "A" {
			resBody["reason"] = reason
			return 8012
		} else {
			return 8013
		}
	}

	// 인증코드를 생성한다
	code := common.GetCodeNumber(6)
	if reqBody["type"].(string) == "phone" && reqBody["phone"].(string) == "000012340000567" {
		code = "123456"
	}
	if reqBody["type"].(string) == "email" && reqBody["email"].(string) == "obsrapptest@obsr.org" {
		code = "123456"
	}

	// 에러카운트가 있으면 가져온다
	var errorCount int
	if loginInfo != nil && loginInfo["errcnt"] != nil {
		errorCount = (int)(loginInfo["errcnt"].(float64))
	}

	// Redis에 캐싱값을 기록한다
	mapV := map[string]interface{} {"step": "1", "code": code, "expire": g_login_curtime + (int64)(global.SendCodeExpireSecs * 1000), "errcnt": errorCount, "userkey": userkey}
	jsonStr, _ := json.Marshal(mapV)
	if _, err = rds.Do("HSET", "DataStore:SendCode:Login", g_login_hashkey, jsonStr); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 인증코드를 전송한다
	if reqBody["type"].(string) == "phone" {
		if reqBody["phone"].(string) != "000012340000567" {
			//if _, err = common.SMSApi_Send(reqBody["ncode"].(string), reqBody["phone"].(string), "Login", code); err != nil {
			//	global.FLog.Println(err)
			//	return 9901
			//}
		}
	} else {
		if _, err = common.MailApi_SendMail(reqBody["email"].(string), "Login", code); err != nil {
			global.FLog.Println(err)
			return 9901
		}
	}

	// 응답값을 세팅한다
	resBody["expire"] = global.SendCodeExpireSecs
	resBody["code"] = code

	return 0
}

// ReqData - step: 2
//         - type: Request Type (phone, email)
//         - ncode: type == phone, 국가코드
//         - phone: type == phone, 핸드폰
//         - email: type == email, 이메일
// ResData - info: 사용자 정보
//         - wallet: 지갑정보
//         - key: 사용자키
//         - errcnt: 오류횟수
//         - maxerr: 최대 오류횟수
//         - limit_time: 제한시간 (timestamp)
func _DSLoginStep2(ctx context.Context, db *sql.DB, rds redis.Conn, reqBody map[string]interface{}, resBody map[string]interface{}, loginInfo map[string]interface{}) int {
	
	// check input
	if reqBody["code"] == nil { return 9003 }

	// check cache
	if loginInfo == nil || loginInfo["step"] == nil || loginInfo["step"].(string) != "1" { return 9902 }
	if loginInfo["code"] == nil || loginInfo["expire"] == nil { return 9902 }

	// 타임아웃을 체크
	if g_login_curtime > (int64)(loginInfo["expire"].(float64)) { return 8007 }

	var err error
	var rvalue string
	var jsonStr []byte

	// 코드를 체크한다
	if reqBody["code"].(string) != loginInfo["code"].(string) {

		errorCount := (int)(loginInfo["errcnt"].(float64)) + 1

		if errorCount < global.SendCodeMaxErrors {  // 오류횟수가 최대허용횟수 미만이라면
			loginInfo["errcnt"] = errorCount
			jsonStr, _ = json.Marshal(loginInfo)
			if _, err = rds.Do("HSET", "DataStore:SendCode:Login", g_login_hashkey, jsonStr); err != nil {
				global.FLog.Println(err)
				return 9901
			}

			resBody["errcnt"] = errorCount
			resBody["maxerr"] = global.SendCodeMaxErrors
			return 8008

		} else {		// 오류횟수가 최대허용횟수 이상이라면
			blockTime := g_login_curtime + (int64)(global.SendCodeBlockSecs * 1000)
			rvalue = `{"block_time": ` + strconv.FormatInt(blockTime, 10) + `}`
			if _, err = rds.Do("HSET", "DataStore:SendCode:Login", g_login_hashkey, rvalue); err != nil {
				global.FLog.Println(err)
				return 9901
			}

			resBody["errcnt"] = errorCount
			resBody["maxerr"] = global.SendCodeMaxErrors
			resBody["limit_time"] = blockTime
			return 8005
		}
	}

	// 로그인을 처리한
	if _, err = common.DSUser_Login(rds, loginInfo["userkey"].(string)); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 로그인 정보를 가져온다
	var mapUser map[string]interface{}
	if mapUser, err = common.DSUser_GetInfo(ctx, db, rds, loginInfo["userkey"].(string)); err != nil {
		global.FLog.Println(err)
		return 9901
	}
	global.FLog.Println(mapUser);

	// 응답값을 세팅한다
	resBody["userkey"] = mapUser["userkey"].(string)
	resBody["loginkey"] = mapUser["loginkey"].(string)
	resBody["info"] = mapUser["info"].(map[string]interface{})
	resBody["wallet"] = mapUser["wallet"].([]map[string]interface{})
	
	// 캐시 정보는 삭제한다
	if _, err = rds.Do("HDEL", "DataStore:SendCode:Login", g_login_hashkey); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	return 0
}
