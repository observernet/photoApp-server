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

var g_update_user_curtime int64

// ReqData - 
// ResData - 
func TR_UpdateUser(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	ctx, cancel := context.WithTimeout(c, global.DBContextTimeout * time.Second)
	defer cancel()

	userkey := reqData["key"].(string)
	reqBody := reqData["body"].(map[string]interface{})
	
	// check input
	if reqBody["loginkey"] == nil { return 9003 }
	if reqBody["step"] == nil { return 9003 }
	g_update_user_curtime = time.Now().UnixNano() / 1000000
	
	var err error

	// 유저 정보를 가져온다
	var mapUser map[string]interface{}
	if mapUser, err = common.User_GetInfo(rds, userkey); err != nil {
		if err == redis.ErrNil {
			return 8015
		} else {
			global.FLog.Println(err)
			return 9901
		}
	}

	// 계정 상태 및 로그인 정보를 체크한다
	if mapUser["info"].(map[string]interface{})["STATUS"].(string) != "V" { return 8013 }
	if mapUser["login"].(map[string]interface{})["loginkey"].(string) != reqBody["loginkey"].(string) { return 8014 }

	var rvalue string
	var StepInfo map[string]interface{}

	// Redis에서 캐싱값을 가져온다
	if rvalue, err = redis.String(rds.Do("HGET", global.Config.Service.Name + ":SendCode:UpdateUser", userkey)); err != nil {
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
		case "1": res_code = _UpdateUserStep1(ctx, db, rds, reqBody, resBody, mapUser["info"].(map[string]interface{}), StepInfo)
		case "2": res_code = _UpdateUserStep2(ctx, db, rds, reqBody, resBody, mapUser["info"].(map[string]interface{}), StepInfo)
		default: res_code = 9003
	}
	
	return res_code
}

func _UpdateUserStep1(ctx context.Context, db *sql.DB, rds redis.Conn, reqBody map[string]interface{}, resBody map[string]interface{}, UserInfo map[string]interface{}, StepInfo map[string]interface{}) int {

	// check input
	if reqBody["type"] == nil { return 9003 }
	if reqBody["type"].(string) != "phone" && reqBody["type"].(string) != "email" && reqBody["type"].(string) != "name" { return 9003 }
	if reqBody["type"].(string) == "phone" && (reqBody["ncode"] == nil || reqBody["phone"] == nil) { return 9003 }
	if reqBody["type"].(string) == "phone" && (len(reqBody["ncode"].(string)) == 0 || len(reqBody["phone"].(string)) == 0) { return 9003 }
	if reqBody["type"].(string) == "phone" && string(reqBody["ncode"].(string)[0]) == "+" { return 9003 }
	if reqBody["type"].(string) == "email" && (reqBody["email"] == nil || len(reqBody["email"].(string)) == 0 ) { return 9003 }
	if reqBody["type"].(string) == "name" && (reqBody["name"] == nil || len(reqBody["name"].(string)) == 0 ) { return 9003 }

	// 인증번호 5회 이상 실패인지 확인한다
	if StepInfo != nil && StepInfo["block_time"] != nil {
		blockTime := (int64)(StepInfo["block_time"].(float64))
		if g_update_user_curtime <= blockTime {
			resBody["limit_time"] = blockTime
			return 8005
		}
	}

	// 인증코드를 생성한다
	code := common.GetCodeNumber(6)

	// 에러카운트가 있으면 가져온다
	var errorCount int
	if StepInfo != nil && StepInfo["errcnt"] != nil {
		errorCount = (int)(StepInfo["errcnt"].(float64))
	}

	if reqBody["type"].(string) == "phone" {

		// 이전값과 동일한지 체크한다
		if reqBody["ncode"].(string) == UserInfo["NCODE"].(string) && reqBody["phone"].(string) == UserInfo["PHONE"].(string) {
			return 8027
		}

		// 이미 존재하는 휴대전화인지 체크한다
		var userCount int64
		query := "SELECT count(USER_KEY) FROM USER_INFO WHERE NCODE = '" + reqBody["ncode"].(string) + "' and PHONE = '" + reqBody["phone"].(string) + "' and STATUS <> 'C'"
		if err := db.QueryRowContext(ctx, query).Scan(&userCount); err != nil {
			if err == sql.ErrNoRows {
				userCount = 0
			} else {
				global.FLog.Println(err)
				return 9901
			}
		}
		if userCount > 0 { return 8001 }
		
		// Redis에 캐싱값을 기록한다
		mapV := map[string]interface{} {"step": "1", "code": code, "expire": g_update_user_curtime + (int64)(global.SendCodeExpireSecs * 1000), "errcnt": errorCount, "type": reqBody["type"].(string), "ncode": reqBody["ncode"].(string), "phone": reqBody["phone"].(string)}
		jsonStr, _ := json.Marshal(mapV)
		if _, err := rds.Do("HSET", global.Config.Service.Name + ":SendCode:UpdateUser", UserInfo["USER_KEY"].(string), jsonStr); err != nil {
			global.FLog.Println(err)
			return 9901
		}

		// 인증코드를 전송한다
		 _, err := common.SMSApi_Send(reqBody["ncode"].(string), reqBody["phone"].(string), "UpdateUser", code)
		 if err != nil {
			global.FLog.Println(err)
			return 9901
		}

	} else if reqBody["type"].(string) == "email" {

		// 이전값과 동일한지 체크한다
		if strings.EqualFold(reqBody["email"].(string), UserInfo["EMAIL"].(string)) {
			return 8027
		}

		// 이미 존재하는 이메일 인지 체크한다
		var userCount int64
		query := "SELECT count(USER_KEY) FROM USER_INFO WHERE UPPER(EMAIL) = '" + strings.ToUpper(reqBody["email"].(string)) + "' and STATUS <> 'C'"
		if err := db.QueryRowContext(ctx, query).Scan(&userCount); err != nil {
			if err == sql.ErrNoRows {
				userCount = 0
			} else {
				global.FLog.Println(err)
				return 9901
			}
		}
		if userCount > 0 { return 8003 }

		// Redis에 캐싱값을 기록한다
		mapV := map[string]interface{} {"step": "1", "code": code, "expire": g_update_user_curtime + (int64)(global.SendCodeExpireSecs * 1000), "errcnt": errorCount, "type": reqBody["type"].(string), "email": reqBody["email"].(string)}
		jsonStr, _ := json.Marshal(mapV)
		if _, err := rds.Do("HSET", global.Config.Service.Name + ":SendCode:UpdateUser", UserInfo["USER_KEY"].(string), jsonStr); err != nil {
			global.FLog.Println(err)
			return 9901
		}

		// 인증코드를 전송한다
		_, err := common.MailApi_SendMail(reqBody["email"].(string), "UpdateUser", code)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}

	} else {

		// 이전값과 동일한지 체크한다
		if strings.EqualFold(reqBody["name"].(string), UserInfo["NAME"].(string)) {
			return 8027
		}

		// 닉네임이 이미 존재하는지 체크한다
		var count int64
		err := db.QueryRowContext(ctx, "SELECT count(USER_KEY) FROM USER_INFO WHERE UPPER(NAME) = '" + strings.ToUpper(reqBody["name"].(string)) + "' and STATUS <> 'C'").Scan(&count)
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
		pass, err := common.CheckForbiddenWord(ctx, db, "N", reqBody["name"].(string))
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}
		if ( pass == true ) { return 8023 }

		// Redis에 캐싱값을 기록한다
		mapV := map[string]interface{} {"step": "1", "code": code, "expire": g_update_user_curtime + (int64)(global.SendCodeExpireSecs * 1000), "errcnt": errorCount, "type": reqBody["type"].(string), "name": reqBody["name"].(string)}
		jsonStr, _ := json.Marshal(mapV)
		if _, err := rds.Do("HSET", global.Config.Service.Name + ":SendCode:UpdateUser", UserInfo["USER_KEY"].(string), jsonStr); err != nil {
			global.FLog.Println(err)
			return 9901
		}

		// 인증코드를 전송한다
		if _, err = common.SMSApi_Send(UserInfo["NCODE"].(string), UserInfo["PHONE"].(string), "UpdateUser", code); err != nil {
			global.FLog.Println(err)
			return 9901
		}

	}

	// 응답값을 세팅한다
	resBody["expire"] = global.SendCodeExpireSecs
	//resBody["code"] = code

	return 0
}

func _UpdateUserStep2(ctx context.Context, db *sql.DB, rds redis.Conn, reqBody map[string]interface{}, resBody map[string]interface{}, UserInfo map[string]interface{}, StepInfo map[string]interface{}) int {
	
	// check input
	if reqBody["code"] == nil { return 9003 }

	// check cache
	if StepInfo == nil || StepInfo["step"] == nil || StepInfo["step"].(string) != "1" { return 9902 }
	if StepInfo["code"] == nil || StepInfo["expire"] == nil { return 9902 }

	// 타임아웃을 체크
	if g_update_user_curtime > (int64)(StepInfo["expire"].(float64)) { return 8007 }

	var err error
	var rvalue string
	var jsonStr []byte

	// 코드를 체크한다
	if reqBody["code"].(string) != StepInfo["code"].(string) {

		errorCount := (int)(StepInfo["errcnt"].(float64)) + 1

		if errorCount < global.SendCodeMaxErrors {  // 오류횟수가 최대허용횟수 미만이라면
			StepInfo["errcnt"] = errorCount
			jsonStr, _ = json.Marshal(StepInfo)
			if _, err = rds.Do("HSET", global.Config.Service.Name + ":SendCode:UpdateUser", UserInfo["USER_KEY"].(string), jsonStr); err != nil {
				global.FLog.Println(err)
				return 9901
			}

			resBody["errcnt"] = errorCount
			resBody["maxerr"] = global.SendCodeMaxErrors
			return 8008

		} else {		// 오류횟수가 최대허용횟수 이상이라면
			blockTime := g_update_user_curtime + (int64)(global.SendCodeBlockSecs * 1000)
			rvalue = `{"block_time": ` + strconv.FormatInt(blockTime, 10) + `}`
			if _, err = rds.Do("HSET", global.Config.Service.Name + ":SendCode:UpdateUser", UserInfo["USER_KEY"].(string), rvalue); err != nil {
				global.FLog.Println(err)
				return 9901
			}

			resBody["errcnt"] = errorCount
			resBody["maxerr"] = global.SendCodeMaxErrors
			resBody["limit_time"] = blockTime
			return 8005
		}
	}

	if StepInfo["type"].(string) == "phone" {

		// 휴대폰 정보를 갱신한다
		query := "UPDATE USER_INFO SET NCODE = '" + StepInfo["ncode"].(string) + "', PHONE = '" + StepInfo["phone"].(string) + "', UPDATE_TIME = sysdate WHERE USER_KEY = '" + UserInfo["USER_KEY"].(string) + "'"
		_, err = db.ExecContext(ctx, query)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}
	} else if StepInfo["type"].(string) == "email" {

		// 이메일 정보를 갱신한다
		query := "UPDATE USER_INFO SET EMAIL = '" + StepInfo["email"].(string) + "', UPDATE_TIME = sysdate WHERE USER_KEY = '" + UserInfo["USER_KEY"].(string) + "'"
		_, err = db.ExecContext(ctx, query)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}
	} else {
		
		// 이름 정보를 갱신한다
		query := "UPDATE USER_INFO SET NAME = '" + StepInfo["name"].(string) + "', UPDATE_TIME = sysdate WHERE USER_KEY = '" + UserInfo["USER_KEY"].(string) + "'"
		_, err = db.ExecContext(ctx, query)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}
	}

	// REDIS 사용자 정보를 갱신한다
	if err = common.User_UpdateInfo(ctx, db, rds, UserInfo["USER_KEY"].(string)); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 캐시 정보는 삭제한다
	if _, err = rds.Do("HDEL", global.Config.Service.Name + ":SendCode:UpdateUser", UserInfo["USER_KEY"].(string)); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 응답값을 세팅한다
	resBody["ok"] = true

	return 0
}
