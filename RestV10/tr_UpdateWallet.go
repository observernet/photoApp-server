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

// ReqData - 
// ResData - 
func TR_UpdateWallet(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

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

	// 현재 지갑의 수량을 체크한다
	wallet_count := len(mapUser["wallet"].([]map[string]interface{}))
	if wallet_count == 0 { return 8203 }
	if wallet_count > 1 { return 8215 }

	// 현재 지갑의 타입을 체크한다
	if mapUser["wallet"].([]map[string]interface{})[0]["WALLET_TYPE"].(string) != "C" { return 8216 }

	// 지갑주소를 생성해서 가져온다
	var address string
	if address, err = common.KAS_CreateAccount(userkey); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	var tx *sql.Tx

	// 트랜잭션 시작
	if tx, err = db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable}); err != nil {
		global.FLog.Println(err)
		return 9901
	}
	defer tx.Rollback()

	// 기존주소는 사용안함으로 변경한다
	_, err = tx.Exec("UPDATE WALLET_INFO SET IS_USE = 'N' WHERE USER_KEY = '" + userkey + "'")
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 새로운 주소를 등록한다
	_, err = tx.Exec("INSERT INTO WALLET_INFO " +
					" (ADDRESS, WALLET_TYPE, CERT_INFO, USER_KEY, IS_USE, UPDATE_TIME) " +
					" VALUES " +
					" ('" + address + "', 'K', '" + global.Config.Service.AccountPool + "', '" + userkey + "', 'Y', sysdate) ")					 
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 트랜잭션 종료
	if err = tx.Commit(); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// REDIS 지갑 정보를 갱신한다
	if err = common.User_UpdateWallet(ctx, db, rds, userkey); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 유저 정보를 다시 가져온다
	if mapUser, err = common.User_GetInfo(rds, userkey); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 응답값을 세팅한다
	wallets := make([]map[string]interface{}, 0)
	if mapUser["wallet"] != nil {
		for _, wallet := range mapUser["wallet"].([]map[string]interface{}) {
			wallets = append(wallets, map[string]interface{} {
											"address": wallet["ADDRESS"].(string),
											"type":    wallet["WALLET_TYPE"].(string),
											"name":    wallet["NAME"].(string)})
		}
	}
	resBody["wallet"] = wallets

	return 0
}
