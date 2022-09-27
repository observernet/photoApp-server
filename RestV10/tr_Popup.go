package RestV10

import (	
	"photoApp-server/global"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

// ReqData - 
// ResData -
func TR_Popup(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	//reqBody := reqData["body"].(map[string]interface{})

	// 팝업 리스트를 가져온다
	query := "SELECT TITLE, BODY, LANG, IS_VISIBLE " +
			 "FROM MAIN_POPUP "
	rows, err := db.Query(query)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	defer rows.Close()

	var title, body, slang, is_visible string

	list := make([]map[string]interface{}, 0)
	for rows.Next() {	
		err = rows.Scan(&title, &body, &slang, &is_visible)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}
		
		list = append(list, map[string]interface{} {"title": title, "body": body, "lang": slang, "visible": is_visible})
	}

	resBody["list"] = list

	return 0
}
