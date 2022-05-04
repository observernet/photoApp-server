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

var g_curtime int64
var g_phoneNumber string

func TR_Join(db *sql.DB, rds redis.Conn, reqData map[string]interface{}, resBody map[string]interface{}) int {

	reqBody := reqData["body"].(map[string]interface{})
	
	// check input
	if reqBody["step"] == nil || reqBody["ncode"] == nil || reqBody["phone"] == nil { return 9003 }
	if reqBody["ncode"] != nil && string(reqBody["ncode"].(string)[0]) == "+" { return 9003 }

	// global variable
	g_curtime = time.Now().UnixNano() / 1000000
	g_phoneNumber = common.GetPhoneNumber(reqBody["ncode"].(string), reqBody["phone"].(string))

	var err error
	var rkey, rvalue string
	var joinInfo map[string]interface{}

	// Redis에서 캐싱값을 가져온다
	rkey = "Join:" + g_phoneNumber
	if rvalue, err = redis.String(rds.Do("GET", rkey)); err != nil {
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
		default: res_code = 9003
	}
	
	return res_code
}

// ReqData - step: 1
//         - ncode: 국가코드
//         - phone: 전화번호
//         - force: true / false
// ResData - expire: 만료시간 (초)
//         - limit_time: 제한시간 (timestamp)
//         - code: 인증코드 (6자리) - 임시, 오픈시 삭제할 예정임
func _JoinStep1(db *sql.DB, rds redis.Conn, reqBody map[string]interface{}, resBody map[string]interface{}, joinInfo map[string]interface{}) int {

	// check input
	if reqBody["force"] == nil { return 9003 }

	// 인증번호 5회 이상 실패인지 확인한다
	if joinInfo != nil && joinInfo["block_time"] != nil {
		blockTime := (int64)(joinInfo["block_time"].(float64))
		if g_curtime <= blockTime {
			resBody["limit_time"] = blockTime
			return 8005
		}
	}

	var err error
	var stmt *sql.Stmt
	var query string
	var userCount int

	// 이미 존재하는 계정인지 체크한다
	query = "SELECT count(USER_KEY) FROM USER_INFO WHERE NCODE = :1 and PHONE = :2 and STATUS <> 'C'";
	if stmt, err = db.Prepare(query); err != nil {
		global.FLog.Println(err)
		return 9901
	}
	defer stmt.Close()

	if err = stmt.QueryRow(reqBody["ncode"].(string), reqBody["phone"].(string)).Scan(&userCount); err != nil {
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

	// 인증코드를 생성한다
	code := common.GetCodeNumber(6)

	// 에러카운트가 있으면 가져온다
	var errorCount int
	if joinInfo != nil && joinInfo["errcnt"] != nil {
		errorCount = (int)(joinInfo["errcnt"].(float64))
	}

	// Redis에 캐싱값을 기록한다
	rkey := "Join:" + g_phoneNumber
	mapV := map[string]interface{} {"step": "1", "code": code, "expire": g_curtime + (int64)(global.SendCodeExpireSecs * 1000), "errcnt": errorCount, "force": reqBody["force"].(bool)}
	jsonStr, _ := json.Marshal(mapV)
	if _, err = rds.Do("SET", rkey, jsonStr); err != nil {
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
	if g_curtime > (int64)(joinInfo["expire"].(float64)) { return 8007 }

	var err error
	var rkey, rvalue string
	var jsonStr []byte

	// 코드를 체크한다
	rkey = "Join:" + g_phoneNumber
	if reqBody["code"].(string) != joinInfo["code"].(string) {

		errorCount := (int)(joinInfo["errcnt"].(float64)) + 1

		if errorCount < global.SendCodeMaxErrors {  // 오류횟수가 최대허용횟수 미만이라면
			joinInfo["errcnt"] = errorCount
			jsonStr, _ = json.Marshal(joinInfo)
			if _, err = rds.Do("SET", rkey, jsonStr); err != nil {
				global.FLog.Println(err)
				return 9901
			}

			resBody["ok"] = false
			resBody["errcnt"] = errorCount
			resBody["maxerr"] = global.SendCodeMaxErrors
			return 8008

		} else {		// 오류횟수가 최대허용횟수 이상이라면
			blockTime := g_curtime + (int64)(global.SendCodeBlockSecs * 1000)
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

	// Redis에 캐싱값을 기록한다
	mapV := map[string]interface{} {"step": "2", "force": joinInfo["force"].(bool)}
	jsonStr, _ = json.Marshal(mapV)
	if _, err = rds.Do("SET", rkey, jsonStr); err != nil {
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
//         - loginpw: 로그인 비밀번호
// ResData - userkey: 사용자 고유키
func _JoinStep3(db *sql.DB, rds redis.Conn, reqBody map[string]interface{}, resBody map[string]interface{}, joinInfo map[string]interface{}) int {
	
	// check input
	if reqBody["name"] == nil || len(reqBody["name"].(string)) <= 1 { return 9003 }
	if reqBody["loginpw"] == nil || len(reqBody["loginpw"].(string)) <= 1 { return 9003 }
	
	// check cache
	if joinInfo == nil || joinInfo["step"] == nil || joinInfo["step"].(string) != "2" { return 9902 }

	//////////////////////////////////////////
	// 회원가입을 처리한다

	var err error
	var tx *sql.Tx
	var userkey string

	// 유저키를 가져온다
	if userkey, err = _JoinGetUserKey(db); err != nil {
		global.FLog.Println(err)
		return 9901
	}
	global.FLog.Println(userkey)

	// 지갑주소를 생성해서 가져온다
	cert_info := global.Config.Service.AccountPool
	//common.InquiryCallToKASConn(userkey)

	address := "0x2Ff9EDC9faCc1738ff1563cA425D219D810078e7"

	// 트랜잭션 시작
	tx, err = db.Begin()
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	defer tx.Rollback()

	// 강제처리라면, 이전의 전화번호를 지워준다
	if joinInfo["force"] != nil && joinInfo["force"].(bool) == true {
		_, err = tx.Exec("UPDATE USER_INFO SET STATUS = 'C', CLOSE_TIME = sysdate, UPDATE_TIME = sysdate WHERE NCODE = :1 and PHONE = :2", reqBody["ncode"].(string), reqBody["phone"].(string))
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}
	}

	// 회원가입을 처리한다
	_, err = tx.Exec("INSERT INTO USER_INFO " +
					 " (USER_KEY, NCODE, PHONE, NAME, LOGIN_PASSWD, USER_LEVEL, STATUS, CREATE_TIME, ERROR_COUNT, UPDATE_TIME) " +
					 " VALUES " +
					 " (:1, :2, :3, :4, :5, 1, 'V', sysdate, 0, sysdate) ",
					 userkey, reqBody["ncode"].(string), reqBody["phone"].(string), reqBody["name"].(string), reqBody["loginpw"].(string))
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 지갑주소를 삽입한다
	_, err = tx.Exec("INSERT INTO WALLET_INFO " +
					 " (ADDRESS, WALLET_TYPE, CERT_INFO, USER_KEY, UPDATE_TIME) " +
					 " VALUES " +
					 " (:1, 'K', :2, :3, sysdate) ",
					 address, cert_info, userkey)					 
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 트랜잭션 종료
	err = tx.Commit()
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 여기까지 회원가입 처리
	//////////////////////////////////////////

	// 응답값을 세팅한다
	resBody["userkey"] = userkey

	return 0
}

func _JoinGetUserKey(db *sql.DB) (string, error) {

	var err error
	var stmt *sql.Stmt
	var userKey string
	var userCount int

	for {
		userKey = common.GetCodeKey(16)

		query := "SELECT count(USER_KEY) FROM USER_INFO WHERE USER_KEY = :1"
		if stmt, err = db.Prepare(query); err != nil {
			return "", err
		}
		defer stmt.Close()

		if err = stmt.QueryRow(userKey).Scan(&userCount); err != nil {
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