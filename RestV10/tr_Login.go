package RestV10

import (
	"time"

	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gomodule/redigo/redis"
)

// ReqData - type: Request Type (phone, email)
//         - ncode: type == phone, 국가코드
//         - phone: type == phone, 핸드폰
//         - email: type == email, 이메일
//         - loginpw: 로그인 비번
// ResData - info: 사용자 정보
//         - wallet: 지갑정보
//         - limit_time: 로그인 제한 시간
//         - errcnt: 오류횟수
//         - reason: 정책위반사유
func TR_Login(db *sql.DB, rds redis.Conn, reqData map[string]interface{}, resBody map[string]interface{}) int {

	reqBody := reqData["body"].(map[string]interface{})
	
	// check input
	if reqBody["type"] == nil || (reqBody["type"].(string) != "phone" && reqBody["type"].(string) != "email") { return 9003 }
	if reqBody["type"].(string) == "phone" && (reqBody["ncode"] == nil || reqBody["phone"] == nil) { return 9003 }
	if reqBody["type"].(string) == "email" && reqBody["email"] == nil { return 9003 }
	if reqBody["ncode"] != nil && string(reqBody["ncode"].(string)[0]) == "+" { return 9003 }
	if reqBody["loginpw"] == nil { return 9003 }
	curtime := time.Now().UnixNano() / 1000000

	var err error
	var stmt *sql.Stmt
	var query string
	var userkey, loginpw, status, reason string
	var error_count int
	var login_limit_time int64

	// 계정정보를 가져온다
	if reqBody["type"].(string) == "phone" {
		query = "SELECT USER_KEY, LOGIN_PASSWD, STATUS, ERROR_COUNT, ABUSE_REASON, NVL(LOGIN_LIMIT_TIME, 0) FROM USER_INFO WHERE NCODE = :1 and PHONE = :2";
		if stmt, err = db.Prepare(query); err != nil {
			global.FLog.Println(err)
			return 9901
		}
		defer stmt.Close()

		if err = stmt.QueryRow(reqBody["ncode"].(string), reqBody["phone"].(string)).Scan(&userkey, &loginpw, &status, &error_count, &reason, &login_limit_time); err != nil {
			if err == sql.ErrNoRows {
				return 8010
			} else {
				global.FLog.Println(err)
				return 9901
			}
		}
	} else {
		query = "SELECT USER_KEY, LOGIN_PASSWD, STATUS, ERROR_COUNT, ABUSE_REASON, NVL(LOGIN_LIMIT_TIME, 0) FROM USER_INFO WHERE EMAIL = :1";
		if stmt, err = db.Prepare(query); err != nil {
			global.FLog.Println(err)
			return 9901
		}
		defer stmt.Close()

		if err = stmt.QueryRow(reqBody["email"].(string)).Scan(&userkey, &loginpw, &status, &error_count, &reason, &login_limit_time); err != nil {
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

	// 오류횟수를 체크한다
	if error_count >= global.LoginMaxErrors {
		if login_limit_time > curtime {
			resBody["limit_time"] = login_limit_time
			return 8011
		} else {
			error_count = 0
		}
	}

	// 비밀번호를 체크한다
	if reqBody["loginpw"].(string) != loginpw {
		error_count = error_count + 1

		if error_count >= global.LoginMaxErrors {
			login_limit_time = curtime + (int64)(global.LoginBlockSecs * 1000)
			_, err = db.Exec("UPDATE USER_INFO SET ERROR_COUNT = :1, LOGIN_LIMIT_TIME = :2 WHERE USER_KEY = :3", error_count, login_limit_time, userkey)
			if err != nil {
				global.FLog.Println(err)
				return 9901
			}

			resBody["limit_time"] = login_limit_time
			return 8011
		} else {
			_, err = db.Exec("UPDATE USER_INFO SET ERROR_COUNT = :1 WHERE USER_KEY = :2", error_count, userkey)
			if err != nil {
				global.FLog.Println(err)
				return 9901
			}

			resBody["errcnt"] = error_count
			return 8010
		}
	} else {
		_, err = db.Exec("UPDATE USER_INFO SET ERROR_COUNT = 0, LOGIN_LIMIT_TIME = 0 WHERE USER_KEY = :1", userkey)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}
	}

	// 로그인을 처리한다
	if _, err = common.User_Login(db, rds, userkey); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 로그인 정보를 가져온다
	var mapUser map[string]interface{}
	if mapUser, err = common.User_GetInfo(rds, userkey); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 응답값을 세팅한다
	resBody["key"] = mapUser["info"].(map[string]interface{})["USER_KEY"].(string)
	resBody["info"] = map[string]interface{} {
							"ncode": mapUser["info"].(map[string]interface{})["NCODE"].(string),
							"phone": mapUser["info"].(map[string]interface{})["PHONE"].(string),
							"email": mapUser["info"].(map[string]interface{})["EMAIL"].(string),
							"name":  mapUser["info"].(map[string]interface{})["NAME"].(string),
							"photo": mapUser["info"].(map[string]interface{})["PHOTO"].(string),
							"level": mapUser["info"].(map[string]interface{})["USER_LEVEL"].(float64)}

	wallets := make([]map[string]interface{}, 0)
	if mapUser["wallet"] != nil {
		for _, wallet := range mapUser["wallet"].([]interface{}) {
			wallets = append(wallets, map[string]interface{} {
											"address": wallet.(map[string]interface{})["ADDRESS"].(string),
											"type":    wallet.(map[string]interface{})["WALLET_TYPE"].(string),
											"name":    wallet.(map[string]interface{})["NAME"].(string)})
		}
	}
	resBody["wallet"] = wallets

	return 0
}
