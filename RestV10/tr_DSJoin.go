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

var g_DSjoin_curtime int64
var g_DSjoin_hashkey string

func TR_DSJoin(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	ctx, cancel := context.WithTimeout(c, global.DBContextTimeout * time.Second)
	defer cancel()

	// Check Header
	if reqData["comm"] == nil || reqData["comm"].(string) != global.Comm_DataStore {
		return 9005
	}
	reqBody := reqData["body"].(map[string]interface{})
	
	// check input
	if reqBody["step"] == nil { return 9003 }
	if reqBody["ncode"] == nil || reqBody["phone"] == nil { return 9003 }
	if reqBody["ncode"] != nil && string(reqBody["ncode"].(string)[0]) == "+" { return 9003 }

	// Global 변수값을 세팅한다
	g_DSjoin_curtime = time.Now().UnixNano() / 1000000
	g_DSjoin_hashkey = common.GetPhoneNumber(reqBody["ncode"].(string), reqBody["phone"].(string))

	var err error
	var rvalue string
	var joinInfo map[string]interface{}

	// Redis에서 캐싱값을 가져온다
	if rvalue, err = redis.String(rds.Do("HGET", "DataStore:SendCode:Join", g_DSjoin_hashkey)); err != nil {
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
		case "1": res_code = _DSJoinStep1(ctx, db, rds, reqBody, resBody, joinInfo)
		case "2": res_code = _DSJoinStep2(ctx, db, rds, reqBody, resBody, joinInfo)
		default: res_code = 9003
	}
	
	return res_code
}

//
// ReqData - step: 1
//         - ncode: type == phone, 국가코드
//         - phone: type == phone, 핸드폰
// ResData - expire: 만료시간 (초)
//         - limit_time: 제한시간 (timestamp)
//         - reason: 정책위반사유
//         - code: 인증코드 (6자리) - 임시, 오픈시 삭제할 예정임
func _DSJoinStep1(ctx context.Context, db *sql.DB, rds redis.Conn, reqBody map[string]interface{}, resBody map[string]interface{}, joinInfo map[string]interface{}) int {

	if reqBody["name"] == nil { return 9003 }

	// 인증번호 5회 이상 실패인지 확인한다
	if joinInfo != nil && joinInfo["block_time"] != nil {
		blockTime := (int64)(joinInfo["block_time"].(float64))
		if g_DSjoin_curtime <= blockTime {
			resBody["limit_time"] = blockTime
			return 8005
		}
	}

	// 에러카운트가 있으면 가져온다
	var errorCount int
	if joinInfo != nil && joinInfo["errcnt"] != nil {
		errorCount = (int)(joinInfo["errcnt"].(float64))
	}

	var userCount int64

	// 계정정보이 존재하는지 체크한다
	query := "SELECT count(USER_KEY) FROM USER_INFO WHERE NCODE = :1 and PHONE = :2";
	if err := db.QueryRowContext(ctx, query, reqBody["ncode"].(string), reqBody["phone"].(string)).Scan(&userCount); err != nil {
		global.FLog.Println(err)
		return 9901
	}
	if userCount > 0 {
		return 8001
	}

	// 인증코드를 생성한다
	code := common.GetCodeNumber(6)

	// Redis에 캐싱값을 기록한다
	mapV := map[string]interface{} {"step": "1", "code": code, "expire": g_DSjoin_curtime + (int64)(global.SendCodeExpireSecs * 1000), "errcnt": errorCount, "name": reqBody["name"].(string)}
	jsonStr, _ := json.Marshal(mapV)
	if _, err := rds.Do("HSET", "DataStore:SendCode:Join", g_DSjoin_hashkey, jsonStr); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 인증코드를 전송한다
	//if _, err = common.SMSApi_Send(reqBody["ncode"].(string), reqBody["phone"].(string), "Join", code); err != nil {
	//	global.FLog.Println(err)
	//	return 9901
	//}

	// 응답값을 세팅한다
	resBody["expire"] = global.SendCodeExpireSecs
	resBody["code"] = code

	return 0
}

// ReqData - step: 2
//         - ncode: type == phone, 국가코드
//         - phone: type == phone, 핸드폰
// ResData - info: 사용자 정보
//         - wallet: 지갑정보
//         - key: 사용자키
//         - errcnt: 오류횟수
//         - maxerr: 최대 오류횟수
//         - limit_time: 제한시간 (timestamp)
func _DSJoinStep2(ctx context.Context, db *sql.DB, rds redis.Conn, reqBody map[string]interface{}, resBody map[string]interface{}, joinInfo map[string]interface{}) int {
	
	// check input
	if reqBody["code"] == nil { return 9003 }

	// check cache
	if joinInfo == nil || joinInfo["step"] == nil || joinInfo["step"].(string) != "1" { return 9902 }
	if joinInfo["code"] == nil || joinInfo["expire"] == nil { return 9902 }

	// 타임아웃을 체크
	if g_DSjoin_curtime > (int64)(joinInfo["expire"].(float64)) { return 8007 }

	var err error
	var rvalue string
	var jsonStr []byte

	// 코드를 체크한다
	if reqBody["code"].(string) != joinInfo["code"].(string) {

		errorCount := (int)(joinInfo["errcnt"].(float64)) + 1

		if errorCount < global.SendCodeMaxErrors {  // 오류횟수가 최대허용횟수 미만이라면
			joinInfo["errcnt"] = errorCount
			jsonStr, _ = json.Marshal(joinInfo)
			if _, err = rds.Do("HSET", "DataStore:SendCode:Join", g_DSjoin_hashkey, jsonStr); err != nil {
				global.FLog.Println(err)
				return 9901
			}

			resBody["errcnt"] = errorCount
			resBody["maxerr"] = global.SendCodeMaxErrors
			return 8008

		} else {		// 오류횟수가 최대허용횟수 이상이라면
			blockTime := g_DSjoin_curtime + (int64)(global.SendCodeBlockSecs * 1000)
			rvalue = `{"block_time": ` + strconv.FormatInt(blockTime, 10) + `}`
			if _, err = rds.Do("HSET", "DataStore:SendCode:Join", g_DSjoin_hashkey, rvalue); err != nil {
				global.FLog.Println(err)
				return 9901
			}

			resBody["errcnt"] = errorCount
			resBody["maxerr"] = global.SendCodeMaxErrors
			resBody["limit_time"] = blockTime
			return 8005
		}
	}

	// 캐시 정보는 삭제한다
	if _, err = rds.Do("HDEL", "DataStore:SendCode:Join", g_DSjoin_hashkey); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	
	var tx *sql.Tx
	var userkey, address string

	// 유저키를 가져온다
	if userkey, err = _DSJoinGetUserKey(ctx, db); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 지갑주소를 생성해서 가져온다
	if address, err = common.KAS_CreateAccount(userkey); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 트랜잭션 시작
	if tx, err = db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable}); err != nil {
		global.FLog.Println(err)
		return 9901
	}
	defer tx.Rollback()

	// 회원가입을 처리한다
	_, err = tx.Exec("INSERT INTO USER_INFO " +
		" (USER_KEY, NCODE, PHONE, NAME, PROMOTION, USER_LEVEL, STATUS, CREATE_TIME, LABEL_COUNT, ERROR_COUNT, LAST_SNAP_TIME, MAIN_LANG, DATASTORE_AGREE, UPDATE_TIME) " +
		" VALUES " +
		" ('" + userkey + "', '" + reqBody["ncode"].(string) + "', '" + reqBody["phone"].(string) + "', '" + joinInfo["name"].(string) + "', 'N', 1, 'V', sysdate, 10, 0, 0, 'E', 'Y', sysdate) ")
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


	// 로그인을 처리한
	if _, err = common.DSUser_Login(rds, userkey); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 로그인 정보를 가져온다
	var mapUser map[string]interface{}
	if mapUser, err = common.DSUser_GetInfo(ctx, db, rds, userkey); err != nil {
		global.FLog.Println(err)
		return 9901
	}
	global.FLog.Println(mapUser);

	// 응답값을 세팅한다
	resBody["userkey"] = mapUser["userkey"].(string)
	resBody["info"] = mapUser["info"].(map[string]interface{})
	resBody["wallet"] = mapUser["wallet"].([]map[string]interface{})

	return 0
}

func _DSJoinGetUserKey(ctx context.Context, db *sql.DB) (string, error) {

	var err error
	//var stmt *sql.Stmt
	var userKey string
	var userCount int

	for {
		userKey = common.GetCodeKey(16)

		if err = db.QueryRowContext(ctx, "SELECT count(USER_KEY) FROM USER_INFO WHERE USER_KEY = '" + userKey + "'").Scan(&userCount); err != nil {
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
