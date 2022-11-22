package RestV10

import (
	"time"
	"context"

	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

// ReqData - 
// ResData - 
func TR_WSList(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	ctx, cancel := context.WithTimeout(c, global.DBContextTimeout * time.Second)
	defer cancel()

	userkey := reqData["key"].(string)
	reqBody := reqData["body"].(map[string]interface{})
	
	// check input
	if reqBody["loginkey"] == nil { return 9003 }

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

	// 계정 상태를 체크한다
	if mapUser["info"].(map[string]interface{})["STATUS"].(string) != "V" {
		if mapUser["info"].(map[string]interface{})["STATUS"].(string) == "A" {
			resBody["reason"] = mapUser["info"].(map[string]interface{})["ABUSE_REASON"].(string)
			return 8012
		} else {
			return 8013
		}
	}

	// 로그인정보가 일치하는지 체크
	if mapUser["login"].(map[string]interface{})["loginkey"].(string) != reqBody["loginkey"].(string) {
		return 8014
	}

	// 웨더스테이션 리스트를 가져온다
	rows, err := db.QueryContext(ctx, "SELECT SERIAL_NO FROM USER_MWS_INFO WHERE USER_KEY = '" + userkey + "' and IS_USE = 'Y'")
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	defer rows.Close()

	var WSSerail, WSSerialList string
	for rows.Next() {	
		err = rows.Scan(&WSSerail)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}

		WSSerialList = WSSerialList + WSSerail + ","
	}

	// WS의 주소정보를 가져와 세팅한다
	WSlist := make([]map[string]interface{}, 0)
	if len(WSSerialList) > 0 {
		WSSerialList = WSSerialList[0:len(WSSerialList)-1]
		WSAPiRes, err := common.WSApi_GetWSData(WSSerialList)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}
		//global.FLog.Println(WSAPiRes)

		for _, mapWS := range WSAPiRes["body"].([]interface{}) {
			WSlist = append(WSlist, map[string]interface{} {
				"serial": mapWS.(map[string]interface {})["serial"].(string),
				"name": mapWS.(map[string]interface {})["name"].(string),
				"address": mapWS.(map[string]interface {})["address"].(string),
				"time": (int64)(mapWS.(map[string]interface {})["time"].(float64)) * 1000})
		}
	}
	resBody["wsList"] = WSlist

	return 0
}
