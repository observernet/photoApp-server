package RestV10

import (
	"fmt"
	"time"
	"context"
	//"encoding/json"
	
	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

// ReqData - 
// ResData - 
func TR_InoutList(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

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

	// 계정 상태 및 로그인 정보를 체크한다
	if mapUser["info"].(map[string]interface{})["STATUS"].(string) != "V" { return 8013 }
	if mapUser["login"].(map[string]interface{})["loginkey"].(string) != reqBody["loginkey"].(string) { return 8014 }

	// 입출금 내역을 가져온다
	query := "SELECT INOUT_IDX, INOUT_TYPE, DATE_TO_UNIXTIME(INOUT_DATE), INOUT_AMOUNT, FROM_ADDRESS, TO_ADDRESS, TX_HASH, PROC_STATUS " +
			 "FROM INOUT_OBSR " +
			 "WHERE USER_KEY = '" + userkey + "' "
	if reqBody["next"] != nil && len(reqBody["next"].(string)) > 0 { query = query + "  and INOUT_IDX < " + reqBody["next"].(string) + " " }
	query = query + "ORDER BY INOUT_IDX desc "
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	defer rows.Close()

	var inout_idx, inout_date int64
	var inout_type, from, to, txHash, status string
	var inout_amount float64

	var count int64

	list := make([]map[string]interface{}, 0)
	for rows.Next() {	
		err = rows.Scan(&inout_idx, &inout_type, &inout_date, &inout_amount, &from, &to, &txHash, &status)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}
		list = append(list, map[string]interface{} {
					"type": inout_type,
					"amount": common.RoundFloat64(inout_amount, global.OBSR_PDesz),
					"from": from,
					"to": to,
					"txHash": txHash,
					"time": inout_date * 1000,
					"status": status})
		
		count = count + 1
		if count >= 30 {
			resBody["next"] = fmt.Sprintf("%d", inout_idx)
			break
		}
	}
	resBody["list"] = list

	return 0
}
