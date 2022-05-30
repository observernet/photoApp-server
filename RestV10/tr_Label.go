package RestV10

import (
	"encoding/json"
	
	"photoApp-server/global"

	"database/sql"
	"github.com/gomodule/redigo/redis"
)

// ReqData - 
// ResData - android: 안드로이드버전
//         - ios: IOS버전
//		   - web: 웹버전
func TR_Label(db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	//reqBody := reqData["body"].(map[string]interface{})

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
