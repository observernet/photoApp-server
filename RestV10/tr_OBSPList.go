package RestV10

import (
	//"encoding/json"
	
	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

// ReqData - 
// ResData - 
func TR_OBSPList(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

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

	// 첫조회라면, OBSP정보를 가져온다
	if reqBody["next"] == nil || len(reqBody["next"].(string)) == 0 {

		var adminVar global.AdminConfig

		// 관리자 정의변수를 가져온다
		if adminVar, err = common.GetAdminVar(rds); err != nil {
			global.FLog.Println(err)
			return 9901
		}

		// OBSP 잔액을 가져온다
		obsp, err := common.GetUserOBSP(db, userkey)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}

		resBody["obsp"] = map[string]interface{} {"curr": obsp, "auto": adminVar.Reword.AutoExchange}
	}

	// 보상/환전 내역을 가져온다
	query := "SELECT D, T, SUM(AMOUNT) " +
			 "FROM " +
			 "( " +
			 "	( " +
			 "		SELECT A.REWORD_DATE D, A.DEVICE T, B.REWORD_AMOUNT AMOUNT " +
			 "		FROM REWORD_LIST A, REWORD_DETAIL B " +
			 "		WHERE A.REWORD_IDX = B.REWORD_IDX " +
			 "		  and A.PROC_STATUS = 'V' " +
			 "		  and A.REWORD_IDX > 0 " +
			 "		  and B.USER_KEY = '" + userkey + "' " +
			 "	) " +
			 "	UNION ALL " +
			 "	( " +
			 "		SELECT TO_NUMBER(TO_CHAR(REQ_TIME, 'RRRRMMDD')) D, 'E' T, PROC_AMOUNT AMOUNT " +
			 "		FROM EXCHANGE_OBSP " +
			 "		WHERE PROC_STATUS = 'V' " +
			 "		and USER_KEY = '" + userkey + "' " +
			 "	) " +
			 ") " +
			 "GROUP BY D, T " +
			 "ORDER BY D desc, T desc";
	rows, err := db.Query(query)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	defer rows.Close()

	var list_count, row_date, prev_date int64
	var row_type string
	var row_amount float64

	list := make([]map[string]interface{}, 0)
	list_date := make([]map[string]interface{}, 0)
	for rows.Next() {	
		err = rows.Scan(&row_date, &row_type, &row_amount)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}

		if prev_date != row_date {
			if len(list_date) > 0 {
				list = append(list, map[string]interface{} {"date": prev_date, "data": list_date})
			}
			list_count = list_count + 1
			if list_count >= 29 { break }

			list_date = make([]map[string]interface{}, 0)
		}

		list_date = append(list_date, map[string]interface{} {"type": row_type, "amount": row_amount})
		prev_date = row_date
	}
	if len(list_date) > 0 {
		list = append(list, map[string]interface{} {"date": prev_date, "data": list_date})
	}
	resBody["list"] = list

	return 0
}
