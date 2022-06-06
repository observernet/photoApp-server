package RestV10

import (
	"encoding/json"
	
	"photoApp-server/global"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

// ReqData - 
// ResData - list: 배너리스트 {title: 제목, image: 이미지, link: 이동링크}
func TR_Banner(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	//reqBody := reqData["body"].(map[string]interface{})

	// 배너 정보를 가져온다
	rkey := global.Config.Service.Name + ":Banner"
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
	lists := make([]map[string]interface{}, 0)
	if mapV["list"] != nil {
		for _, lst := range mapV["list"].([]interface{}) {
			if lst.(map[string]interface{})["LANG"].(string) == lang {
				lists = append(lists, map[string]interface{} {
					"title": lst.(map[string]interface{})["TITLE"].(string),
					"image": lst.(map[string]interface{})["IMAGE"].(string),
					"link": lst.(map[string]interface{})["LINK"].(string)})
			}
		}
	}
	resBody["list"] = lists

	return 0
}
