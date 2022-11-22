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

var g_login_curtime int64
var g_login_hashkey string

func TR_Login(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

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
	if rvalue, err = redis.String(rds.Do("HGET", global.Config.Service.Name + ":SendCode:Login", g_login_hashkey)); err != nil {
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
		case "1": res_code = _LoginStep1(ctx, db, rds, reqBody, resBody, loginInfo)
		case "2": res_code = _LoginStep2(ctx, db, rds, reqBody, resBody, loginInfo)
		default: res_code = 9003
	}
	
	return res_code
}

// ReqData - step: 1
//         - type: Request Type (phone, email)
//         - ncode: type == phone, 국가코드
//         - phone: type == phone, 핸드폰
//         - email: type == email, 이메일
// ResData - expire: 만료시간 (초)
//         - limit_time: 제한시간 (timestamp)
//         - reason: 정책위반사유
//         - code: 인증코드 (6자리) - 임시, 오픈시 삭제할 예정임
func _LoginStep1(ctx context.Context, db *sql.DB, rds redis.Conn, reqBody map[string]interface{}, resBody map[string]interface{}, loginInfo map[string]interface{}) int {

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
	if _, err = rds.Do("HSET", global.Config.Service.Name + ":SendCode:Login", g_login_hashkey, jsonStr); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 인증코드를 전송한다
	if reqBody["type"].(string) == "phone" {
		if reqBody["phone"].(string) != "000012340000567" {
			if _, err = common.SMSApi_Send(reqBody["ncode"].(string), reqBody["phone"].(string), "Login", code); err != nil {
				global.FLog.Println(err)
				return 9901
			}
		}
	} else {
		if _, err = common.MailApi_SendMail(reqBody["email"].(string), "Login", code); err != nil {
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
//         - ncode: type == phone, 국가코드
//         - phone: type == phone, 핸드폰
//         - email: type == email, 이메일
// ResData - info: 사용자 정보
//         - wallet: 지갑정보
//         - key: 사용자키
//         - errcnt: 오류횟수
//         - maxerr: 최대 오류횟수
//         - limit_time: 제한시간 (timestamp)
func _LoginStep2(ctx context.Context, db *sql.DB, rds redis.Conn, reqBody map[string]interface{}, resBody map[string]interface{}, loginInfo map[string]interface{}) int {
	
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
			if _, err = rds.Do("HSET", global.Config.Service.Name + ":SendCode:Login", g_login_hashkey, jsonStr); err != nil {
				global.FLog.Println(err)
				return 9901
			}

			resBody["errcnt"] = errorCount
			resBody["maxerr"] = global.SendCodeMaxErrors
			return 8008

		} else {		// 오류횟수가 최대허용횟수 이상이라면
			blockTime := g_login_curtime + (int64)(global.SendCodeBlockSecs * 1000)
			rvalue = `{"block_time": ` + strconv.FormatInt(blockTime, 10) + `}`
			if _, err = rds.Do("HSET", global.Config.Service.Name + ":SendCode:Login", g_login_hashkey, rvalue); err != nil {
				global.FLog.Println(err)
				return 9901
			}

			resBody["errcnt"] = errorCount
			resBody["maxerr"] = global.SendCodeMaxErrors
			resBody["limit_time"] = blockTime
			return 8005
		}
	}

	// 로그인을 처리한다
	var loginkey string
	if loginkey, err = common.User_Login(ctx, db, rds, loginInfo["userkey"].(string)); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 로그인 정보를 가져온다
	var mapUser map[string]interface{}
	if mapUser, err = common.User_GetInfo(rds, loginInfo["userkey"].(string)); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 관리자변수를 가져온다
	var adminVar global.AdminConfig
	if adminVar, err = common.GetAdminVar(rds); err != nil {
		global.FLog.Println(err)
		return 9901
	}
	remain_snap_time := g_login_curtime - (int64)(mapUser["stat"].(map[string]interface{})["LAST_SNAP_TIME"].(float64))
	remain_snap_time = adminVar.Snap.Interval - remain_snap_time / 1000
	if remain_snap_time < 0 { remain_snap_time = 0 }

	// 퍼소나 등록내역이 있는지 체크한다
	var persona_count int64
	err = db.QueryRowContext(ctx, "SELECT count(USER_KEY) FROM USER_PERSONA WHERE USER_KEY = '" + loginInfo["userkey"].(string) + "'").Scan(&persona_count)
	if err != nil {
		if err == sql.ErrNoRows {
			persona_count = 0
		} else {
			global.FLog.Println(err)
			return 9901
		}
	}

	// 응답값을 세팅한다
	resBody["userkey"] = loginInfo["userkey"].(string)
	resBody["loginkey"] = loginkey
	resBody["info"] = map[string]interface{} {
							"ncode": mapUser["info"].(map[string]interface{})["NCODE"].(string),
							"phone": mapUser["info"].(map[string]interface{})["PHONE"].(string),
							"email": mapUser["info"].(map[string]interface{})["EMAIL"].(string),
							"name":  mapUser["info"].(map[string]interface{})["NAME"].(string),
							"photo": "https://photoapp.obsr-app.org/Image/View/profile/" + loginInfo["userkey"].(string),
							"level": mapUser["info"].(map[string]interface{})["USER_LEVEL"].(float64),
							"promotion": mapUser["info"].(map[string]interface{})["PROMOTION"].(string),
							"persona_cnt": persona_count}
	
	resBody["stat"] = map[string]interface{} {
							"obsp": common.RoundFloat64(mapUser["stat"].(map[string]interface{})["OBSP"].(float64), global.OBSR_PDesz),
							"labels": mapUser["stat"].(map[string]interface{})["LABEL_COUNT"].(float64),
							"remain_snap_time": remain_snap_time,
							"snap_interval": adminVar.Snap.Interval,
							"count": map[string]interface{} {
								"snap": mapUser["stat"].(map[string]interface{})["TODAY_SNAP_COUNT"].(float64),
								"snap_rp": mapUser["stat"].(map[string]interface{})["TODAY_SNAP_COUNT"].(float64) * adminVar.Reword.Snap,
								"label": mapUser["stat"].(map[string]interface{})["TODAY_LABEL_COUNT"].(float64),
								"label_rp": mapUser["stat"].(map[string]interface{})["TODAY_LABEL_COUNT"].(float64) * adminVar.Reword.Label,
								"label_etc": mapUser["stat"].(map[string]interface{})["TODAY_LABEL_ETC_COUNT"].(float64),
								"label_etc_rp": mapUser["stat"].(map[string]interface{})["TODAY_LABEL_ETC_COUNT"].(float64) * adminVar.Reword.LabelEtc}}

	resBody["reword"] = map[string]interface{} {
							"snap": adminVar.Reword.Snap,
							"label": adminVar.Reword.Label,
							"label_etc": adminVar.Reword.LabelEtc,
							"obsp_per_day": adminVar.Reword.OBSPPerDay,
							"auto_exchange": adminVar.Reword.AutoExchange,
							"persona": adminVar.Reword.Persona}

	wallets := make([]map[string]interface{}, 0)
	if mapUser["wallet"] != nil {
		for _, wallet := range mapUser["wallet"].([]map[string]interface{}) {
			wallets = append(wallets, map[string]interface{} {
											"address": wallet["ADDRESS"].(string),
											"type":    wallet["WALLET_TYPE"].(string),
											"name":    wallet["NAME"].(string)})
		}
	}
	resBody["wallet"] = wallets

	// 캐시 정보는 삭제한다
	if _, err = rds.Do("HDEL", global.Config.Service.Name + ":SendCode:Login", g_login_hashkey); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	return 0
}
