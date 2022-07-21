package RestV10

import (
	"fmt"
	"strings"
	"encoding/json"
	
	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

// ReqData - 
// ResData - 
func TR_Withdraw(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	userkey := reqData["key"].(string)
	reqBody := reqData["body"].(map[string]interface{})

	// check input
	if reqBody["loginkey"] == nil || reqBody["from"] == nil || reqBody["to"] == nil || reqBody["amount"] == nil { return 9003 }

	// Time Check
	curtime := common.GetIntTime()
	if curtime <= 2000 || curtime >= 235000 { return 8211 }

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

	// from주소가 고객주소인지 체크한다
	var wallet_type, wallet_cert string
	if len(mapUser["wallet"].([]map[string]interface{})) > 0 {
		var isSearch bool
		for _, wallet := range mapUser["wallet"].([]map[string]interface{}) {
			if strings.EqualFold(wallet["ADDRESS"].(string), reqBody["from"].(string)) {
				wallet_type = wallet["WALLET_TYPE"].(string)
				wallet_cert = wallet["CERT_INFO"].(string)
				isSearch = true
				break
			}
		}
		if isSearch == false { return 8206 }
	} else {
		return 8203
	}
	if wallet_type != "K" { return 8214	}

	// to 주소를 체크한다
	if len(reqBody["to"].(string)) != 42 { return 8021 }

	// 처리중인 출금내역이 있는지 체크한다
	var count_prev_request int64
	err = db.QueryRow("SELECT count(WITHDRAW_IDX) FROM WITHDRAW_OBSR WHERE USER_KEY = '" + userkey + "' and PROC_STATUS = 'A'").Scan(&count_prev_request)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	if count_prev_request > 0 { return 8212 }

	// 수수료 면제 티켓을 체크한다
	if reqBody["fee_ticket"] != nil && reqBody["fee_ticket"].(bool) == true {
		global.FLog.Println("수수료 면제 티켓이 없습니다")
		return 8201
	}

	// OBSR 잔액을 가져온다
	balance, err := common.KAS_GetBalanceOf(userkey, reqBody["from"].(string), wallet_type)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	obsr := common.GetFloat64FromString(balance)

	// 예상 수수료를 계산한다
	mapFee, err := common.GetTxFee(rds, adminVar.TxFee.Withdraw.Coin, adminVar.TxFee.Withdraw.Fee)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	txfee := mapFee["txfee"].(float64)

	// 출금가능금액을 체크한다
	if (reqBody["amount"].(float64) + txfee) > obsr {
		global.FLog.Println("출금 가능 금액을 초과하였습니다 [%s]", userkey)
		return 8213
	}
	
	// 출금 내역을 DB에 기록한다
	withdraw_idx, err := _WithdrawInsertDB(db, userkey, reqBody["amount"].(float64), reqBody["from"].(string), reqBody["to"].(string), txfee)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// KASConn에 출금을 요청한다
	var mapKAS map[string]interface{}
	kas, err := common.KAS_Transfer("W:" + userkey, reqBody["from"].(string), reqBody["to"].(string), fmt.Sprintf("%f", reqBody["amount"].(float64)), wallet_type, wallet_cert)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	if err = json.Unmarshal([]byte(kas), &mapKAS); err != nil { return 9901 }

	// KASConn 응답값에 따라 출금 내역을 갱신한다 (중복 출금을 막기 위해 KASConn 요청을 기준으로 2단계로 처리함)
	if mapKAS["success"].(bool) == true {

		// 환전 내역에 데이타키를 세팅한다
		query := fmt.Sprintf("UPDATE WITHDRAW_OBSR SET KASCONN_KEY = '%d' WHERE WITHDRAW_IDX = %d", (int64)(mapKAS["msg"].(float64)), withdraw_idx)
		_, err = db.Exec(query)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}

		// KASConn에 수수료 출금을 요청한다
		kas, err = common.KAS_Transfer("F:" + userkey, reqBody["from"].(string), adminVar.Wallet.Withdraw.Address, fmt.Sprintf("%f", txfee), wallet_type, wallet_cert)
		if err == nil {
			if err = json.Unmarshal([]byte(kas), &mapKAS); err == nil {
				global.FLog.Println(mapKAS)
			} else {
				global.FLog.Println(err)
			}
		} else {
			global.FLog.Println(err)
		}
	} else {

		// 환전을 거부처리한다
		query := fmt.Sprintf("UPDATE WITHDRAW_OBSR SET PROC_TIME = sysdate, PROC_AMOUNT = 0, WITHDRAW_FEE = 0, PROC_STATUS = 'Z', MEMO = '%s', UPDATE_TIME = sysdate " +
							 "WHERE WITHDRAW_IDX = %d",
							 mapKAS["msg"].(string), withdraw_idx)
		_, err = db.Exec(query)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}

		resBody["msg"] = mapKAS["msg"].(string)
	}

	// 사용자에게 응답을 전송한다
	resBody["key"] = withdraw_idx
	resBody["from"] = reqBody["from"].(string)
	resBody["to"] = reqBody["to"].(string)
	resBody["amount"] = reqBody["amount"].(float64)
	resBody["txfee"] = txfee

	return 0
}

func _WithdrawInsertDB(db *sql.DB, userkey string, amount float64, from string, to string, txfee float64) (int64, error) {

	var err error
	var withdraw_idx int64

	// 환전키를 가져온다
	err = db.QueryRow("SELECT NVL(MAX(WITHDRAW_IDX), 0) + 1 FROM WITHDRAW_OBSR").Scan(&withdraw_idx)
	if err != nil { return 0, err }

	// 출금내역을 저장한다 (Auto commit)
	query := fmt.Sprintf("INSERT INTO WITHDRAW_OBSR (WITHDRAW_IDX, USER_KEY, REQ_TYPE, REQ_TIME, REQ_AMOUNT, FROM_ADDRESS, TO_ADDRESS, WITHDRAW_FEE, PROC_STATUS, UPDATE_TIME) " +
						 "VALUES (%d, '%s', 'U', sysdate, %f, '%s', '%s', %f, 'A', sysdate) ",
						 withdraw_idx, userkey, amount, from, to, txfee)
	_, err = db.Exec(query)					 
	if err != nil { return 0, err }

	return withdraw_idx, nil
}
