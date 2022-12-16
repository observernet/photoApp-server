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


func TR_Reaction(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	ctx, cancel := context.WithTimeout(c, global.DBContextTimeout * time.Second)
	defer cancel()

	userkey := reqData["key"].(string)
	reqBody := reqData["body"].(map[string]interface{})

	// check input
	if reqBody["loginkey"] == nil || reqBody["snapkey"] == nil { return 9003 }
	if reqBody["type"] == nil || (reqBody["type"].(string) != "L") { return 9003 }

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

	// 스냅키를 가져온다
	snap_date, snap_idx, err := common.GetSnapKey(reqBody["snapkey"].(string))
	if err != nil {
		global.FLog.Println(err)
		return 8104
	}

	// 해당스냅의 정보를 가져온다
	var row_myReactionIdx int64
	var row_user_key, row_is_show, row_upload_status string
	query := "SELECT USER_KEY, IS_SHOW, UPLOAD_STATUS, " +
			 "       NVL((SELECT REACTION_IDX FROM SNAP_REACTION WHERE SNAP_DATE = A.SNAP_DATE and SNAP_IDX = A.SNAP_IDX and USER_KEY = '" + userkey + "' AND REACTION_TYPE = '" + reqBody["type"].(string) + "'), 0) " +
	         "FROM SNAP A " +
			 "WHERE SNAP_DATE = " + snap_date + " and SNAP_IDX = " + snap_idx;
	err = db.QueryRowContext(ctx, query).Scan(&row_user_key, &row_is_show, &row_upload_status, &row_myReactionIdx)
	if err != nil {
		if err == sql.ErrNoRows {
			return 8104
		} else {
			global.FLog.Println(err)
			return 9901
		}
	}

	// 스냅정보를 체크한다
	if row_user_key == userkey { return 8110 }
	if row_is_show != "Y" { return 8111 }
	if row_upload_status != "V" { return 8107 }

	//////////////////////////////////////////
	// 라벨 등록 처리한다
	var tx *sql.Tx

	// 트랜잭션 시작
	tx, err = db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	defer tx.Rollback()

	// 리액션 정보를 등록/삭제한다
	if row_myReactionIdx == 0 {

		var reactionIdx int64

		// 리액션키를 가져온다
		query = "SELECT NVL(MAX(REACTION_IDX), 0) + 1 FROM SNAP_REACTION WHERE SNAP_DATE = " + snap_date + " and SNAP_IDX = " + snap_idx;
		err = tx.QueryRow(query).Scan(&reactionIdx)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}

		// 리액션정보를 등록한다
		_, err = tx.Exec("INSERT INTO SNAP_REACTION " +
						 " (SNAP_DATE, SNAP_IDX, REACTION_IDX, USER_KEY, REACTION_TYPE, REACTION_DATE, USER_IP) " + 
						 " VALUES " +
						 " (:1, :2, :3, :4, :5, sysdate, :6) ",
						 snap_date, snap_idx, reactionIdx, userkey, reqBody["type"].(string), c.ClientIP())
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}

	} else {

		// 리액션정보를 삭제한다
		_, err = tx.Exec("DELETE FROM SNAP_REACTION WHERE SNAP_DATE = :1 and SNAP_IDX = :2 and REACTION_IDX = :3 and USER_KEY = :4 ",
						 snap_date, snap_idx, row_myReactionIdx, userkey)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}

	}

	// 트랜잭션 종료
	err = tx.Commit()
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 라벨 등록 처리한다
	//////////////////////////////////////////

	// 응답값을 세팅한다
	if row_myReactionIdx == 0 {
		resBody["proctype"] = "R"
	} else {
		resBody["proctype"] = "C"
	}

	return 0
}
