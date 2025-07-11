package RestV10

import (
	"fmt"
	"time"
	"context"
	"strings"
	"encoding/json"
	
	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

func TR_Exchange(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	ctx, cancel := context.WithTimeout(c, global.DBContextTimeout * time.Second)
	defer cancel()

	userkey := reqData["key"].(string)
	reqBody := reqData["body"].(map[string]interface{})

	// check input
	if reqBody["loginkey"] == nil || reqBody["to"] == nil || reqBody["amount"] == nil { return 9003 }

	// Time Check
	curtime := common.GetIntTime()
	if curtime <= 2000 || curtime >= 235000 { return 8205 }

	var err error

	// 관리자 정의변수를 가져온다
	var adminVar global.AdminConfig
	if adminVar, err = common.GetAdminVar(rds); err != nil {
		global.FLog.Println(err)
		return 9901
	}

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

	// 입력주소가 고객주소인지 체크한다
	if len(mapUser["wallet"].([]map[string]interface{})) > 0 {
		var isSearch bool
		for _, wallet := range mapUser["wallet"].([]map[string]interface{}) {
			if strings.EqualFold(wallet["ADDRESS"].(string), reqBody["to"].(string)) {
				isSearch = true
				break
			}
		}
		if isSearch == false { return 8206 }
	} else {
		return 8203
	}

	// 처리중인 환전내역이 있는지 체크한다
	var count_prev_request int64
	err = db.QueryRowContext(ctx, "SELECT count(EXCHANGE_IDX) FROM EXCHANGE_OBSP WHERE USER_KEY = '" + userkey + "' and PROC_STATUS = 'A'").Scan(&count_prev_request)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	if count_prev_request > 0 { return 8204 }

	// 수수료 면제 티켓을 체크한다
	if reqBody["fee_ticket"] != nil && reqBody["fee_ticket"].(bool) == true {
		global.FLog.Println("수수료 면제 티켓이 없습니다")
		return 8201
	}

	// OBSP 잔액을 가져온다
	obsp, err := common.GetUserOBSP(ctx, db, userkey)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 예상 수수료를 계산한다
	mapFee, err := common.GetTxFee(rds, adminVar.TxFee.Exchange.Coin, adminVar.TxFee.Exchange.Fee)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	txfee := mapFee["txfee"].(float64) * 2.0

	// 환전가능금액을 체크한다
	if (reqBody["amount"].(float64) + txfee) > obsp {
		global.FLog.Println("환전 가능 금액을 초과하였습니다 [%s]", userkey)
		return 8202
	}
	
	// 환전 내역을 DB에 기록한다
	exchange_idx, err := _ExchangeInsertDB(ctx, db, userkey, reqBody["amount"].(float64), adminVar.Wallet.Exchange.Address, reqBody["to"].(string), txfee)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// KASConn에 환전을 요청한다
	kas, err := common.KAS_Transfer("E:" + userkey, adminVar.Wallet.Exchange.Address, reqBody["to"].(string), fmt.Sprintf("%f", reqBody["amount"].(float64)), adminVar.Wallet.Exchange.Type, adminVar.Wallet.Exchange.CertInfo)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	var mapKAS map[string]interface{}
	if err = json.Unmarshal([]byte(kas), &mapKAS); err != nil {
		return 9901
	}
	//global.FLog.Println(mapKAS)

	// KASConn 응답값에 따라 환전 내역을 갱신한다 (중복 환전을 막기 위해 KASConn 요청을 기준으로 2단계로 처리함)
	if mapKAS["success"].(bool) == true {

		// 환전 내역에 데이타키를 세팅한다
		query := fmt.Sprintf("UPDATE EXCHANGE_OBSP SET KASCONN_KEY = '%d' WHERE EXCHANGE_IDX = %d", (int64)(mapKAS["msg"].(float64)), exchange_idx)
		_, err = db.ExecContext(ctx, query)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}
	} else {

		// 환전을 거부처리한다
		query := fmt.Sprintf("UPDATE EXCHANGE_OBSP SET PROC_TIME = sysdate, PROC_AMOUNT = 0, EXCHANGE_FEE = 0, PROC_STATUS = 'Z', MEMO = '%s', UPDATE_TIME = sysdate " +
							 "WHERE EXCHANGE_IDX = %d",
							 mapKAS["msg"].(string), exchange_idx)
		_, err = db.ExecContext(ctx, query)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}

		resBody["msg"] = mapKAS["msg"].(string)
	}

	// 사용자에게 응답을 전송한다
	resBody["key"] = exchange_idx
	resBody["to"] = reqBody["to"].(string)
	resBody["amount"] = common.RoundFloat64(reqBody["amount"].(float64), global.OBSR_PDesz)
	resBody["txfee"] = common.RoundFloat64(txfee, global.OBSR_PDesz)

	return 0
}

func _ExchangeInsertDB(ctx context.Context, db *sql.DB, userkey string, amount float64, from string, to string, txfee float64) (int64, error) {

	var err error
	var exchange_idx int64

	// 환전키를 가져온다
	err = db.QueryRowContext(ctx, "SELECT NVL(MAX(EXCHANGE_IDX), 0) + 1 FROM EXCHANGE_OBSP").Scan(&exchange_idx)
	if err != nil { return 0, err }

	// 환전내역을 저장한다 (Auto commit)
	query := fmt.Sprintf("INSERT INTO EXCHANGE_OBSP (EXCHANGE_IDX, USER_KEY, REQ_TYPE, REQ_TIME, REQ_AMOUNT, FROM_ADDRESS, TO_ADDRESS, EXCHANGE_FEE, PROC_STATUS, UPDATE_TIME) " +
						 "VALUES (%d, '%s', 'U', sysdate, %f, '%s', '%s', %f, 'A', sysdate) ",
						 exchange_idx, userkey, amount, from, to, txfee)
	_, err = db.ExecContext(ctx, query)					 
	if err != nil { return 0, err }

	return exchange_idx, nil
}
