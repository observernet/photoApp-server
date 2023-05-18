package RestV10

import (
	//"fmt"
	"encoding/json"
	
	"photoApp-server/global"
	//"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

// ReqData - 
// ResData - android: 안드로이드버전
//         - ios: IOS버전
//		   - web: 웹버전
func TR_Version(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	//reqBody := reqData["body"].(map[string]interface{})

	//fmt.Println( common.SMSApi_Send("62", "83819435002", "111", "123456") )
	//fmt.Println( common.SMSApi_Send("81", "7044967175", "111", "123456") )

	// 버전 정보를 가져온다
	rkey := global.Config.Service.Name + ":Version"
	rvalue, err := redis.String(rds.Do("GET", rkey))
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// Map으로 변환한다
	mapV := make(map[string]interface{})	
	if err = json.Unmarshal([]byte(rvalue), &mapV); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 응답값을 세팅한다
	resBody["android"] = mapV["android"].(string)
	resBody["ios"] = mapV["ios"].(string)
	resBody["web"] = mapV["web"].(string)

	return 0
}
