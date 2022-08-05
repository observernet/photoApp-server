package RestV10

import (
	"fmt"
	
	"photoApp-server/global"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

// ReqData - 
// ResData -
func TR_Notice(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	reqBody := reqData["body"].(map[string]interface{})

	// check input
	if reqBody["last_update_time"] == nil { return 9003 }

	last_update_time := (int64)(reqBody["last_update_time"].(float64) / 1000)

	// 공지사항 리스트를 가져온다
	query := "SELECT IDX, TYPE, TITLE, BODY, LINK, DATE_TO_UNIXTIME(REG_DATE), DATE_TO_UNIXTIME(UPDATE_TIME), SORT " +
			 "FROM NOTICE " + 
			 "WHERE DATE_TO_UNIXTIME(UPDATE_TIME) > " + fmt.Sprintf("%d", last_update_time) +
			 "ORDER BY SORT desc, IDX desc"
	rows, err := db.Query(query)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	defer rows.Close()

	var idx, sort, reg_time, update_time int64
	var ntype, title, body, link string

	list := make([]map[string]interface{}, 0)
	for rows.Next() {	
		err = rows.Scan(&idx, &ntype, &title, &body, &link, &reg_time, &update_time, &sort)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}
		sort = sort * 100000 + idx
		if link == "" { link = fmt.Sprintf("PhotoApp:Notice:%d", idx)}

		list = append(list, map[string]interface{} {"type": ntype, "title": title, "body": body, "link": link, "sort": sort, "time": reg_time})

		if last_update_time < update_time { last_update_time = update_time }
	}

	resBody["list"] = list
	resBody["last_update_time"] = last_update_time * 1000

	return 0
}
