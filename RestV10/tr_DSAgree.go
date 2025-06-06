package RestV10

import (
	"time"
	"context"

	"photoApp-server/global"
	//"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

func TR_DSAgree(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	ctx, cancel := context.WithTimeout(c, global.DBContextTimeout * time.Second)
	defer cancel()

	// Check Header
	if reqData["comm"] == nil || reqData["comm"].(string) != global.Comm_DataStore {
		return 9005
	}
	reqBody := reqData["body"].(map[string]interface{})
	
	// check input
	if reqBody["type"] == nil || (reqBody["type"].(string) != "phone" && reqBody["type"].(string) != "email") { return 9003 }
	if reqBody["type"].(string) == "phone" && (reqBody["ncode"] == nil || reqBody["phone"] == nil) { return 9003 }
	if reqBody["type"].(string) == "email" && reqBody["email"] == nil { return 9003 }
	if reqBody["type"].(string) == "phone" && reqBody["ncode"] != nil && string(reqBody["ncode"].(string)[0]) == "+" { return 9003 }

	var err error
	var stmt *sql.Stmt
	var query string
	var userkey, status, agree string

	// 계정정보를 가져온다
	if reqBody["type"].(string) == "phone" {
		query = "SELECT USER_KEY, STATUS, NVL(DATASTORE_AGREE, 'N') FROM USER_INFO WHERE NCODE = :1 and PHONE = :2 and STATUS <> 'C'";
		if stmt, err = db.PrepareContext(ctx, query); err != nil {
			global.FLog.Println(err)
			return 9901
		}
		defer stmt.Close()

		if err = stmt.QueryRow(reqBody["ncode"].(string), reqBody["phone"].(string)).Scan(&userkey, &status, &agree); err != nil {
			if err == sql.ErrNoRows {
				return 8010
			} else {
				global.FLog.Println(err)
				return 9901
			}
		}
	} else {
		query = "SELECT USER_KEY, STATUS, NVL(DATASTORE_AGREE, 'N') FROM USER_INFO WHERE EMAIL = :1 and STATUS <> 'C'";
		if stmt, err = db.PrepareContext(ctx, query); err != nil {
			global.FLog.Println(err)
			return 9901
		}
		defer stmt.Close()

		if err = stmt.QueryRow(reqBody["email"].(string)).Scan(&userkey, &status, &agree); err != nil {
			if err == sql.ErrNoRows {
				return 8010
			} else {
				global.FLog.Println(err)
				return 9901
			}
		}
	}

	// 계정 상태에 따라
	if status != "V" {
		if status == "A" {
			return 8012
		} else {
			return 8013
		}
	}

	// 데이타스토어 동의 여부를 체크한다
	if agree == "Y" {
		return 8042
	}

	// 회원정보를 갱신한다
	query = "UPDATE USER_INFO SET DATASTORE_AGREE = 'Y' WHERE USER_KEY = :1"
	_, err = db.Exec(query, userkey)					 
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 응답데이타를 세팅한다
	resBody["ok"] = true
	
	return 0
}
