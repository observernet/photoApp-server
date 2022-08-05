package RestV10

import (
	"time"
	"strconv"
	"encoding/json"

	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

var g_joinout_curtime int64

// ReqData - 
// ResData - 
func TR_JoinOut(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	userkey := reqData["key"].(string)
	reqBody := reqData["body"].(map[string]interface{})
	
	// check input
	if reqBody["loginkey"] == nil { return 9003 }
	if reqBody["step"] == nil { return 9003 }
	g_joinout_curtime = time.Now().UnixNano() / 1000000
	
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
	if rvalue, err = redis.String(rds.Do("HGET", global.Config.Service.Name + ":SendCode:JoinOut", userkey)); err != nil {
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
		case "1": res_code = _JoinOutStep1(db, rds, reqBody, resBody, mapUser["info"].(map[string]interface{}), StepInfo)
		case "2": res_code = _JoinOutStep2(db, rds, reqBody, resBody, mapUser["info"].(map[string]interface{}), StepInfo)
		default: res_code = 9003
	}
	
	return res_code
}

func _JoinOutStep1(db *sql.DB, rds redis.Conn, reqBody map[string]interface{}, resBody map[string]interface{}, UserInfo map[string]interface{}, StepInfo map[string]interface{}) int {

	// 인증번호 5회 이상 실패인지 확인한다
	if StepInfo != nil && StepInfo["block_time"] != nil {
		blockTime := (int64)(StepInfo["block_time"].(float64))
		if g_joinout_curtime <= blockTime {
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
		
	// Redis에 캐싱값을 기록한다
	mapV := map[string]interface{} {"step": "1", "code": code, "expire": g_joinout_curtime + (int64)(global.SendCodeExpireSecs * 1000), "errcnt": errorCount}
	jsonStr, _ := json.Marshal(mapV)
	if _, err := rds.Do("HSET", global.Config.Service.Name + ":SendCode:JoinOut", UserInfo["USER_KEY"].(string), jsonStr); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 인증코드를 전송한다
	common.SendCode_Phone(UserInfo["NCODE"].(string), UserInfo["PHONE"].(string), code)

	// 응답값을 세팅한다
	resBody["expire"] = global.SendCodeExpireSecs
	resBody["code"] = code

	return 0
}

func _JoinOutStep2(db *sql.DB, rds redis.Conn, reqBody map[string]interface{}, resBody map[string]interface{}, UserInfo map[string]interface{}, StepInfo map[string]interface{}) int {
	
	// check input
	if reqBody["code"] == nil { return 9003 }

	// check cache
	if StepInfo == nil || StepInfo["step"] == nil || StepInfo["step"].(string) != "1" { return 9902 }
	if StepInfo["code"] == nil || StepInfo["expire"] == nil { return 9902 }

	// 타임아웃을 체크
	if g_joinout_curtime > (int64)(StepInfo["expire"].(float64)) { return 8007 }

	var err error
	var rvalue string
	var jsonStr []byte

	// 코드를 체크한다
	if reqBody["code"].(string) != StepInfo["code"].(string) {

		errorCount := (int)(StepInfo["errcnt"].(float64)) + 1

		if errorCount < global.SendCodeMaxErrors {  // 오류횟수가 최대허용횟수 미만이라면
			StepInfo["errcnt"] = errorCount
			jsonStr, _ = json.Marshal(StepInfo)
			if _, err = rds.Do("HSET", global.Config.Service.Name + ":SendCode:JoinOut", UserInfo["USER_KEY"].(string), jsonStr); err != nil {
				global.FLog.Println(err)
				return 9901
			}

			resBody["errcnt"] = errorCount
			resBody["maxerr"] = global.SendCodeMaxErrors
			return 8008

		} else {		// 오류횟수가 최대허용횟수 이상이라면
			blockTime := g_joinout_curtime + (int64)(global.SendCodeBlockSecs * 1000)
			rvalue = `{"block_time": ` + strconv.FormatInt(blockTime, 10) + `}`
			if _, err = rds.Do("HSET", global.Config.Service.Name + ":SendCode:JoinOut", UserInfo["USER_KEY"].(string), rvalue); err != nil {
				global.FLog.Println(err)
				return 9901
			}

			resBody["errcnt"] = errorCount
			resBody["maxerr"] = global.SendCodeMaxErrors
			resBody["limit_time"] = blockTime
			return 8005
		}
	}

	// 회원 정보를 갱신한다 (해지)
	query := "UPDATE USER_INFO SET STATUS = 'C', CLOSE_TIME = sysdate, UPDATE_TIME = sysdate WHERE USER_KEY = '" + UserInfo["USER_KEY"].(string) + "'"
	_, err = db.Exec(query)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 로그아웃을 처리한다
	if err = common.User_Logout(rds, UserInfo["USER_KEY"].(string)); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 캐시 정보는 삭제한다
	if _, err = rds.Do("HDEL", global.Config.Service.Name + ":SendCode:JoinOut", UserInfo["USER_KEY"].(string)); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 응답값을 세팅한다
	resBody["ok"] = true

	return 0
}
