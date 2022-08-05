package RestV10

import (
	"time"
	"strings"
	"strconv"
	"encoding/json"

	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

var g_join_time int64
var g_join_rkey string
var g_join_phone string

func TR_Join(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	reqBody := reqData["body"].(map[string]interface{})
	
	// check input
	if reqBody["step"] == nil || reqBody["ncode"] == nil || reqBody["phone"] == nil { return 9003 }
	if len(reqBody["ncode"].(string)) == 0 || len(reqBody["phone"].(string)) == 0 { return 9003 }
	if reqBody["ncode"] != nil && string(reqBody["ncode"].(string)[0]) == "+" { return 9003 }

	// global variable
	g_join_time = time.Now().UnixNano() / 1000000
	g_join_rkey = global.Config.Service.Name + ":SendCode:Join"
	g_join_phone = common.GetPhoneNumber(reqBody["ncode"].(string), reqBody["phone"].(string))

	var err error
	var rvalue string
	var joinInfo map[string]interface{}

	// Redis에서 캐싱값을 가져온다
	if rvalue, err = redis.String(rds.Do("HGET", g_join_rkey, g_join_phone)); err != nil {
		if err != redis.ErrNil {
			global.FLog.Println(err)
			return 9901
		}
	}

	// 캐싱데이타가 존재하면 Map 데이타로 변환한다
	if len(rvalue) > 0 {
		joinInfo = make(map[string]interface{})
		if err = json.Unmarshal([]byte(rvalue), &joinInfo); err != nil {
			global.FLog.Println(err)
			return 9901
		}
	}

	var res_code int
	var step string = reqBody["step"].(string)
	switch step {
		case "1": res_code = _JoinStep1(db, rds, reqBody, resBody, joinInfo)
		case "2": res_code = _JoinStep2(db, rds, reqBody, resBody, joinInfo)
		case "3": res_code = _JoinStep3(db, rds, reqBody, resBody, joinInfo)
		case "4": res_code = _JoinStep4(db, rds, reqBody, resBody, joinInfo)
		default: res_code = 9003
	}
	
	return res_code
}

// ReqData - step: 1
//         - ncode: 국가코드
//         - phone: 전화번호
//         - promotion: true / false
//         - force: true / false
// ResData - expire: 만료시간 (초)
//         - limit_time: 제한시간 (timestamp)
//         - code: 인증코드 (6자리) - 임시, 오픈시 삭제할 예정임
func _JoinStep1(db *sql.DB, rds redis.Conn, reqBody map[string]interface{}, resBody map[string]interface{}, joinInfo map[string]interface{}) int {

	// check input
	if reqBody["promotion"] == nil || reqBody["force"] == nil { return 9003 }

	// 인증번호 5회 이상 실패인지 확인한다
	if joinInfo != nil && joinInfo["block_time"] != nil {
		blockTime := (int64)(joinInfo["block_time"].(float64))
		if g_join_time <= blockTime {
			resBody["limit_time"] = blockTime
			return 8005
		}
	}

	var err error
	var userCount int

	// 이미 존재하는 계정인지 체크한다
	if reqBody["force"].(bool) == false {
		query := "SELECT count(USER_KEY) FROM USER_INFO WHERE NCODE = '" + reqBody["ncode"].(string) + "' and PHONE = '" + reqBody["phone"].(string) + "' and STATUS <> 'C'"
		if err = db.QueryRow(query).Scan(&userCount); err != nil {
			if err == sql.ErrNoRows {
				userCount = 0
			} else {
				global.FLog.Println(err)
				return 9901
			}
		}

		if userCount > 0 {
			global.FLog.Println("이미 가입된 전화번호입니다 [%s:%s]", reqBody["ncode"].(string), reqBody["phone"].(string))
			return 8001
		}
	}

	// 인증코드를 생성한다
	code := common.GetCodeNumber(6)

	// 에러카운트가 있으면 가져온다
	var errorCount int
	if joinInfo != nil && joinInfo["errcnt"] != nil {
		errorCount = (int)(joinInfo["errcnt"].(float64))
	}

	// Redis에 캐싱값을 기록한다
	s_promotion := "Y"
	if reqBody["promotion"].(bool) == false { s_promotion = "N" }
	mapV := map[string]interface{} {"step": "1", "code": code, "expire": g_join_time + (int64)(global.SendCodeExpireSecs * 1000), "errcnt": errorCount, "promotion": s_promotion, "force": reqBody["force"].(bool)}
	jsonStr, _ := json.Marshal(mapV)
	if _, err = rds.Do("HSET", g_join_rkey, g_join_phone, jsonStr); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 인증코드를 전송한다
	common.SendCode_Phone(reqBody["ncode"].(string), reqBody["phone"].(string), code)

	// 응답값을 세팅한다
	resBody["expire"] = global.SendCodeExpireSecs
	resBody["code"] = code

	return 0
}

// ReqData - step: 2
//         - ncode: 국가코드
//         - phone: 전화번호
//         - code: 핸드폰으로 전송한 6자리 코드
// ResData - ok: true/false
//         - errcnt: 오류횟수
//         - maxerr: 최대 오류횟수
//         - limit_time: 제한시간 (timestamp)
func _JoinStep2(db *sql.DB, rds redis.Conn, reqBody map[string]interface{}, resBody map[string]interface{}, joinInfo map[string]interface{}) int {
	
	// check input
	if reqBody["code"] == nil { return 9003 }

	// check cache
	if joinInfo == nil || joinInfo["step"] == nil || joinInfo["step"].(string) != "1" { return 9902 }
	if joinInfo["code"] == nil || joinInfo["expire"] == nil { return 9902 }

	// 타임아웃을 체크
	if g_join_time > (int64)(joinInfo["expire"].(float64)) { return 8007 }

	var err error
	var rvalue string
	var jsonStr []byte

	// 코드를 체크한다
	if reqBody["code"].(string) != joinInfo["code"].(string) {

		errorCount := (int)(joinInfo["errcnt"].(float64)) + 1

		if errorCount < global.SendCodeMaxErrors {  // 오류횟수가 최대허용횟수 미만이라면
			joinInfo["errcnt"] = errorCount
			jsonStr, _ = json.Marshal(joinInfo)
			if _, err = rds.Do("HSET", g_join_rkey, g_join_phone, jsonStr); err != nil {
				global.FLog.Println(err)
				return 9901
			}

			resBody["ok"] = false
			resBody["errcnt"] = errorCount
			resBody["maxerr"] = global.SendCodeMaxErrors
			return 8008

		} else {		// 오류횟수가 최대허용횟수 이상이라면
			blockTime := g_join_time + (int64)(global.SendCodeBlockSecs * 1000)
			rvalue = `{"block_time": ` + strconv.FormatInt(blockTime, 10) + `}`
			if _, err = rds.Do("HSET", g_join_rkey, g_join_phone, rvalue); err != nil {
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
	mapV := map[string]interface{} {"step": "2", "promotion": joinInfo["promotion"].(string), "force": joinInfo["force"].(bool)}
	jsonStr, _ = json.Marshal(mapV)
	if _, err = rds.Do("HSET", g_join_rkey, g_join_phone, jsonStr); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 응답값을 세팅한다
	resBody["ok"] = true

	return 0
}

// ReqData - step: 3
//         - ncode: 국가코드
//         - phone: 전화번호
//         - name: 이름
//         - link: true/false
//         - newold: 구버전 O, 신버전 N
//         - email: 연동이메일
// ResData - expire: 만료시간 (초)
//         - limit_time: 제한시간 (timestamp)
//         - code: 인증코드 (6자리) - 임시, 오픈시 삭제할 예정임
//         - userkey: 사용자 고유키 (link가 false 경우)
func _JoinStep3(db *sql.DB, rds redis.Conn, reqBody map[string]interface{}, resBody map[string]interface{}, joinInfo map[string]interface{}) int {
	
	link := "Y"
	if reqBody["link"] == nil || reqBody["link"].(bool) == false { link = "N" }

	// check input
	if reqBody["name"] == nil || len(reqBody["name"].(string)) <= 1 { return 9003 }
	if reqBody["newold"] == nil || (reqBody["newold"].(string) != "N" && reqBody["newold"].(string) != "O") { return 9003 }
	if link == "N" && reqBody["newold"].(string) == "O" { return 9003 }

	// 인증번호 5회 이상 실패인지 확인한다
	if joinInfo != nil && joinInfo["block_time"] != nil {
		blockTime := (int64)(joinInfo["block_time"].(float64))
		if g_join_time <= blockTime {
			resBody["limit_time"] = blockTime
			return 8005
		}
	}

	// check cache
	if joinInfo == nil || joinInfo["step"] == nil || (joinInfo["step"].(string) != "2" && joinInfo["step"].(string) != "3") { return 9902 }

	var err error

	// 닉네임이 이미 존재하는지 체크한다
	var count int64
	err = db.QueryRow("SELECT count(USER_KEY) FROM USER_INFO WHERE NAME = '" + strings.ToUpper(reqBody["name"].(string)) + "' and STATUS <> 'C'").Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			count = 0
		} else {
			global.FLog.Println(err)
			return 9901
		}
	}
	if count > 0 { return 8022 }

	// 닉네임에 금칙어가 있는지 체크한다
	pass, err := common.CheckForbiddenWord(db, "N", reqBody["name"].(string))
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	if ( pass == true ) { return 8023 }


	if link == "N" && reqBody["newold"].(string) == "N" {

		var tx *sql.Tx
		var userkey, address string

		// 유저키를 가져온다
		if userkey, err = _JoinGetUserKey(db); err != nil {
			global.FLog.Println(err)
			return 9901
		}

		// 지갑주소를 생성해서 가져온다
		if address, err = common.KAS_CreateAccount(userkey); err != nil {
			global.FLog.Println(err)
			return 9901
		}

		// 트랜잭션 시작
		if tx, err = db.Begin(); err != nil {
			global.FLog.Println(err)
			return 9901
		}
		defer tx.Rollback()

		// 강제처리라면, 이전의 전화번호는 해지처리한다
		if joinInfo["force"].(bool) == true {
			if _, err = tx.Exec("UPDATE USER_INFO SET STATUS = 'C', CLOSE_TIME = sysdate, UPDATE_TIME = sysdate WHERE NCODE = '" + reqBody["ncode"].(string) + "' and PHONE = '" + reqBody["phone"].(string) + "'"); err != nil {
				global.FLog.Println(err)
				return 9901
			}
		}

		// 회원가입을 처리한다
		_, err = tx.Exec("INSERT INTO USER_INFO " +
						" (USER_KEY, NCODE, PHONE, NAME, PROMOTION, USER_LEVEL, STATUS, CREATE_TIME, LABEL_COUNT, ERROR_COUNT, LAST_SNAP_TIME, UPDATE_TIME) " +
						" VALUES " +
						" ('" + userkey + "', '" + reqBody["ncode"].(string) + "', '" + reqBody["phone"].(string) + "', '" + reqBody["name"].(string) + "', '" + joinInfo["promotion"].(string) + "', 1, 'V', sysdate, 10, 0, 0, sysdate) ")
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}

		// 지갑주소를 삽입한다
		_, err = tx.Exec("INSERT INTO WALLET_INFO " +
						" (ADDRESS, WALLET_TYPE, CERT_INFO, USER_KEY, IS_USE, UPDATE_TIME) " +
						" VALUES " +
						" ('" + address + "', 'K', '" + global.Config.Service.AccountPool + "', '" + userkey + "', 'Y', sysdate) ")					 
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}

		// 트랜잭션 종료
		if err = tx.Commit(); err != nil {
			global.FLog.Println(err)
			return 9901
		}

		// 캐시 정보는 삭제한다
		if _, err = rds.Do("HDEL", g_join_rkey, g_join_phone); err != nil {
			global.FLog.Println(err)
			return 9901
		}

		// 응답값을 세팅한다
		resBody["userkey"] = userkey

	} else {

		// 이미 존재하는 이메일 인지 체크한다
		var userCount int64
		query := "SELECT count(USER_KEY) FROM USER_INFO WHERE UPPER(EMAIL) = '" + strings.ToUpper(reqBody["email"].(string)) + "' and STATUS <> 'C'"
		if err = db.QueryRow(query).Scan(&userCount); err != nil {
			if err == sql.ErrNoRows {
				userCount = 0
			} else {
				global.FLog.Println(err)
				return 9901
			}
		}

		if userCount > 0 {
			global.FLog.Println("이미 가입된 이메일입니다 [%s:%s]", reqBody["email"].(string), reqBody["email"].(string))
			return 8003
		}

		// WSApi 에서도 존재하는 이메일인지 체크한다
		if reqBody["newold"].(string) == "O" {
			_, err := common.WSApi_CheckUserEmail(reqBody["email"].(string))
			if err != nil {
				if strings.Contains(err.Error(), "Not Found") {
					return 8024
				} else if strings.Contains(err.Error(), "not valid") {
					return 8025
				} else {
					global.FLog.Println(err)
					return 9901
				}
			}
			//if wsRes["code"].(float64) != 0 { return 8024 }
		}

		// 인증코드를 생성한다
		code := common.GetCodeNumber(6)

		// 에러카운트가 있으면 가져온다
		var errorCount int
		if joinInfo != nil && joinInfo["errcnt"] != nil {
			errorCount = (int)(joinInfo["errcnt"].(float64))
		}

		// Redis에 캐싱값을 기록한다
		mapV := map[string]interface{} {"step": "3", "code": code, "expire": g_join_time + (int64)(global.SendCodeExpireSecs * 1000), "errcnt": errorCount,
										"name": reqBody["name"].(string), "link": link, "newold": reqBody["newold"].(string), "email": reqBody["email"].(string), "promotion": joinInfo["promotion"].(string), "force": joinInfo["force"].(bool)}
		jsonStr, _ := json.Marshal(mapV)
		if _, err = rds.Do("HSET", g_join_rkey, g_join_phone, jsonStr); err != nil {
			global.FLog.Println(err)
			return 9901
		}

		// 인증코드를 전송한다
		common.SendCode_Email(reqBody["email"].(string), code)

		// 응답값을 세팅한다
		resBody["expire"] = global.SendCodeExpireSecs
		resBody["code"] = code
	}

	return 0
}


// ReqData - step: 4
//         - ncode: 국가코드
//         - phone: 전화번호
//         - email: 연동이메일
//         - code: 이메일로 전송한 6자리 코드
// ResData - userkey: 사용자 고유키
func _JoinStep4(db *sql.DB, rds redis.Conn, reqBody map[string]interface{}, resBody map[string]interface{}, joinInfo map[string]interface{}) int {
	
	// check input
	if reqBody["code"] == nil { return 9003 }

	// check cache
	if joinInfo == nil || joinInfo["step"] == nil || joinInfo["step"].(string) != "3" { return 9902 }
	if joinInfo["code"] == nil || joinInfo["expire"] == nil { return 9902 }
	if joinInfo["email"] == nil || joinInfo["email"].(string) != reqBody["email"].(string) { return 9902 }

	// 타임아웃을 체크
	if g_join_time > (int64)(joinInfo["expire"].(float64)) { return 8007 }

	var err error
	var rvalue string
	var jsonStr []byte

	// 코드를 체크한다
	if reqBody["code"].(string) != joinInfo["code"].(string) {

		errorCount := (int)(joinInfo["errcnt"].(float64)) + 1

		if errorCount < global.SendCodeMaxErrors {  // 오류횟수가 최대허용횟수 미만이라면
			joinInfo["errcnt"] = errorCount
			jsonStr, _ = json.Marshal(joinInfo)
			if _, err = rds.Do("HSET", g_join_rkey, g_join_phone, jsonStr); err != nil {
				global.FLog.Println(err)
				return 9901
			}

			resBody["errcnt"] = errorCount
			resBody["maxerr"] = global.SendCodeMaxErrors
			return 8008

		} else {		// 오류횟수가 최대허용횟수 이상이라면
			blockTime := g_join_time + (int64)(global.SendCodeBlockSecs * 1000)
			rvalue = `{"block_time": ` + strconv.FormatInt(blockTime, 10) + `}`
			if _, err = rds.Do("HSET", g_join_rkey, g_join_phone, rvalue); err != nil {
				global.FLog.Println(err)
				return 9901
			}

			resBody["errcnt"] = errorCount
			resBody["maxerr"] = global.SendCodeMaxErrors
			resBody["limit_time"] = blockTime
			return 8005
		}
	}

	var tx *sql.Tx
	var userkey string
	var address, address_type, address_key string
	var reword_wslist string
	var oldInfo map[string]interface {}

	// WSApi에서 정보를 가져온다
	if joinInfo["newold"].(string) == "O" {
		if oldInfo, err = common.WSApi_GetUserInfo(joinInfo["email"].(string)); err != nil {
			global.FLog.Println(err)
			return 9901
		}
	}

	// 유저키를 가져온다
	if userkey, err = _JoinGetUserKey(db); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 지갑주소를 생성해서 가져온다
	if oldInfo != nil && oldInfo["body"].(map[string]interface {})["ws_wallet"] != nil && len(oldInfo["body"].(map[string]interface {})["ws_wallet"].(string)) > 10 {
		address = oldInfo["body"].(map[string]interface {})["ws_wallet"].(string)
		address_type = "C"
		address_key = ""
	} else {
		if address, err = common.KAS_CreateAccount(userkey); err != nil {
			global.FLog.Println(err)
			return 9901
		}
		address_type = "K"
		address_key = global.Config.Service.AccountPool
	}

	// 트랜잭션 시작
	if tx, err = db.Begin(); err != nil {
		global.FLog.Println(err)
		return 9901
	}
	defer tx.Rollback()

	// 강제처리라면, 이전의 전화번호는 해지처리한다
	if joinInfo["force"].(bool) == true {
		if _, err = tx.Exec("UPDATE USER_INFO SET STATUS = 'C', CLOSE_TIME = sysdate, UPDATE_TIME = sysdate WHERE NCODE = '" + reqBody["ncode"].(string) + "' and PHONE = '" + reqBody["phone"].(string) + "'"); err != nil {
			global.FLog.Println(err)
			return 9901
		}
	}

	// 회원가입을 처리한다
	_, err = tx.Exec("INSERT INTO USER_INFO " +
					" (USER_KEY, NCODE, PHONE, EMAIL, NAME, PROMOTION, USER_LEVEL, STATUS, CREATE_TIME, LABEL_COUNT, ERROR_COUNT, LAST_SNAP_TIME, UPDATE_TIME) " +
					" VALUES " +
					" ('" + userkey + "', '" + reqBody["ncode"].(string) + "', '" + reqBody["phone"].(string) + "', '" + joinInfo["email"].(string) + "', '" + joinInfo["name"].(string) + "', '" + joinInfo["promotion"].(string) + "', 1, 'V', sysdate, 10, 0, 0, sysdate) ")
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 지갑주소를 삽입한다
	_, err = tx.Exec("INSERT INTO WALLET_INFO " +
					" (ADDRESS, WALLET_TYPE, CERT_INFO, USER_KEY, IS_USE, UPDATE_TIME) " +
					" VALUES " +
					" ('" + address + "', '" + address_type + "', '" + address_key + "', '" + userkey + "', 'Y', sysdate) ")					 
	if err != nil {
		global.FLog.Println(err)
		if strings.Contains(err.Error(), "ORA-00001") {
			return 8019
		} else {
			return 9901
		}
	}

	// oldInfo가 존재한다면
	if oldInfo != nil {

		// 포인트를 기록한다
		if oldInfo["body"].(map[string]interface {})["obsp"] != nil && oldInfo["body"].(map[string]interface {})["obsp"].(float64) > 0 {
			_, err = tx.Exec("INSERT INTO REWORD_DETAIL " +
							" (REWORD_IDX, USER_KEY, SERIAL_NO, REWORD_AMOUNT, UPDATE_TIME) " +
							" VALUES " +
							" (0, '" + userkey + "', 0, " + strconv.FormatInt((int64)(oldInfo["body"].(map[string]interface {})["obsp"].(float64)), 10) + ", sysdate) ")					 
			if err != nil {
				global.FLog.Println(err)
				return 9901
			}
		}

		// WS 리스트를 기록한다
		if oldInfo["body"].(map[string]interface {})["ws"] != nil && len(oldInfo["body"].(map[string]interface {})["ws"].([]interface{})) > 0 {
			for _, ws := range oldInfo["body"].(map[string]interface {})["ws"].([]interface{}) {
				_, err = tx.Exec("INSERT INTO USER_MWS_INFO (USER_KEY, SERIAL_NO, REG_TYPE, REG_TIME, IS_USE, UPDATE_TIME) " +
								"VALUES ('" + userkey + "', " + ws.(map[string]interface{})["serial"].(string) + ", 'L', sysdate, 'Y', sysdate) ")					 
				if err != nil {
					global.FLog.Println(err)
					return 9901
				}

				if ws.(map[string]interface{})["series"] != nil && ws.(map[string]interface{})["series"].(string) != "0" {
					_, err = tx.Exec("INSERT INTO MWS_SERIES_SERIAL (SERIES, SERIAL_NO, IS_ACTIVE, UPDATE_TIME) " +
									"VALUES (" + ws.(map[string]interface{})["series"].(string) + ", " + ws.(map[string]interface{})["serial"].(string) + ", 'Y', sysdate) ")					 
					if err != nil {
						global.FLog.Println(err)
						if strings.Contains(err.Error(), "ORA-00001") {
							return 8020
						} else {
							return 9901
						}
					}
					reword_wslist = reword_wslist + ws.(map[string]interface{})["series"].(string) + "-" + ws.(map[string]interface{})["serial"].(string) + ","
				}
			}
		}
	}

	// 트랜잭션 종료
	if err = tx.Commit(); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 이전 WS 보상은 정지한다
	if len(reword_wslist) > 0 {
		reword_wslist = reword_wslist[0:len(reword_wslist)-1]
		if oldInfo, err = common.WSApi_UpdateWSRewordStatus("N", reword_wslist); err != nil {
			global.FLog.Println(err)
			return 9901
		}
	}

	// 캐시 정보는 삭제한다
	if _, err = rds.Do("HDEL", g_join_rkey, g_join_phone); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 응답값을 세팅한다
	resBody["userkey"] = userkey

	return 0
}

func _JoinGetUserKey(db *sql.DB) (string, error) {

	var err error
	//var stmt *sql.Stmt
	var userKey string
	var userCount int

	for {
		userKey = common.GetCodeKey(16)

		if err = db.QueryRow("SELECT count(USER_KEY) FROM USER_INFO WHERE USER_KEY = '" + userKey + "'").Scan(&userCount); err != nil {
			if err == sql.ErrNoRows {
				userCount = 0
			} else {
				return "", err
			}
		}

		if userCount == 0 { break }
	}

	return userKey, nil
}
