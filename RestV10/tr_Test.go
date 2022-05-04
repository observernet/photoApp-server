package RestV10

import (
	"photoApp-server/global"

	"database/sql"
	"github.com/gomodule/redigo/redis"
)

type Column struct {
	Col1	string
	Col2	string
}

// Receive Request & Send Response
func TR_Test(db *sql.DB, rds redis.Conn, reqData map[string]interface{}, resBody map[string]interface{}) int {

	reqBody := reqData["body"].(map[string]interface{})
	
	global.FLog.Println(reqData["trid"].(string))
	global.FLog.Println(reqBody["key"].(string))

	// Redis Test
	reply, err := redis.String(rds.Do("GET", "key"))
	if err != nil {
		global.FLog.Println(err)
	}
	resBody["redis"] = reply

	// DB Test
	rows, err := db.Query("select COLUMN1, COLUMN2 from TEST_TABLE")
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	defer rows.Close()

	var cols []Column
	var col Column
	for i := 0; rows.Next(); i++ {
		rows.Scan(&col.Col1, &col.Col2)

		cols = append(cols, col)
	}
	resBody["db"] = cols
	
	return 0
}
