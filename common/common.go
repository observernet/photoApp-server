package common

import (
	"time"
	"errors"
	"strconv"
	"strings"
	"math"
	"math/rand"
	"encoding/json"

	"photoApp-server/global"

	"database/sql"
	"github.com/gomodule/redigo/redis"
	"github.com/godror/godror"
)

func GetPhoneNumber(ncode string, phone string) string {

	var phone_number string

	if string(ncode[0]) == "+" {
		phone_number = ncode[1:len(ncode)]
	} else {
		phone_number = ncode
	}

	if string(phone[0]) == "0" {
		phone_number = phone_number + phone[1:len(phone)]
	} else {
		phone_number = phone_number + phone
	}

	return phone_number
}

func GetCodeNumber(length int) string {

	var idx int
	var code string = ""
	var source string = "0123456789"

	for i := 0; i < length ; i++ {
		rand.Seed(time.Now().UnixNano())
		idx = rand.Intn(len(source))
		code = code + string(source[idx])
	}

	return code
}

func GetIntDate() (int64) {
	loc, _ := time.LoadLocation(global.Config.Service.Timezone)
	kst := time.Now().In(loc)
	curtime := kst.Format("20060102")
	return GetInt64FromString(curtime)
}

func GetIntTime() (int64) {
	loc, _ := time.LoadLocation(global.Config.Service.Timezone)
	kst := time.Now().In(loc)
	curtime := kst.Format("150405")
	return GetInt64FromString(curtime)
}

func GetCodeKey(length int) string {

	var idx int
	var code string = ""
	var source string = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	for i := 0; i < length ; i++ {
		rand.Seed(time.Now().UnixNano())
		idx = rand.Intn(len(source))
		code = code + string(source[idx])
	}

	return code
}

func GetAdminVar(rds redis.Conn) (global.AdminConfig, error) {

	admVar := global.AdminConfig{}
	
	rkey := global.Config.Service.Name + ":AdminVar"
	rvalue, err := redis.String(rds.Do("GET", rkey))
	if err != nil { return admVar, err }

	err = json.Unmarshal([]byte(rvalue), &admVar)
	if err != nil { return admVar, err }

	return admVar, nil
}

func GetRowsResult(rows *sql.Rows, limit int) ([]map[string]interface{}, error) {

	if rows == nil {
		return nil, errors.New("Rows is null")
	}

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	data := make([]interface{}, len(cols))
	dataPtr := make([]interface{}, len(cols))
	for i, _ := range data {
		dataPtr[i] = &data[i]
	}

	var count int
	var results []map[string]interface{}
	for rows.Next() {	
		err = rows.Scan(dataPtr...)
		if err != nil {
			return nil, err
		}

		result := make(map[string]interface{})
		for i, item := range dataPtr {
			val := item.(*interface{})
			result[cols[i]] = *val
		}
		results = append(results, result)

		count = count + 1
		if limit > 0 && count >= limit {
			break
		}
	}

	return results, nil
}

func GetSnapKey(key string) (string, string, error) {

	if len(key) != 14 {
		return "", "", errors.New("Not Snap Key")
	}

	snapkey, err := strconv.ParseInt(key, 10, 64)
	if err != nil {
		return "", "", err
	}

	snap_date := strconv.FormatInt(snapkey / 1000000, 10)
	snap_idx := strconv.FormatInt(snapkey % 1000000, 10)

	return snap_date, snap_idx, nil
}

func GetInt64FromNumber(num godror.Number) int64 {

	val, err := num.Value()
	if err != nil {
		return 0
	}

	var ret int64
	if ret, err = strconv.ParseInt(val.(string), 10, 64); err != nil {
		return 0
	}

	return ret
}

func GetFloat64FromNumber(num godror.Number) float64 {

	val, err := num.Value()
	if err != nil {
		return 0
	}

	var ret float64
	if ret, err = strconv.ParseFloat(val.(string), 64); err != nil {
		return 0
	}

	return ret
}

func GetInt64FromString(val string) int64 {

	ret, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0
	}

	return ret
}

func GetFloat64FromString(val string) float64 {

	ret, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0
	}

	return ret
}

func RoundFloat64(val float64, pdesz int) float64 {
	mul := math.Pow(10, float64(pdesz))
	return math.Round(val * mul) / mul
}

func GetUserOBSP(db *sql.DB, userkey string) (float64, error) {

	var err error
	var obsp, reword, exchange float64

	err = db.QueryRow("SELECT NVL(SUM(A.REWORD_AMOUNT), 0) FROM REWORD_DETAIL A, REWORD_LIST B WHERE A.REWORD_IDX = B.REWORD_IDX and A.USER_KEY = '" + userkey + "' and B.PROC_STATUS = 'V'").Scan(&reword)
	if err != nil {
		return 0, err
	}

	err = db.QueryRow("SELECT NVL(SUM(PROC_AMOUNT + EXCHANGE_FEE), 0) FROM EXCHANGE_OBSP WHERE USER_KEY = '" + userkey + "' and PROC_STATUS = 'V'").Scan(&exchange)
	if err != nil {
		return 0, err
	}

	obsp = reword - exchange
	if obsp < 0 { obsp = 0 }

	return obsp, nil
}
/*
func GetTxFeeOBSP(db *sql.DB, klay_txfee float64) (float64, float64, int64, float64, int64, error) {

	var err error
	var rows *sql.Rows
	var symbol string
	var price, obsr_price, klay_price float64
	var price_time, obsr_time, klay_time int64

	if rows, err = db.Query("SELECT SYMBOL, PRICE, PRICE_TIME FROM EXCH_PRICE WHERE SYMBOL in ('OBSR', 'KLAY')"); err != nil {
		return 0, 0, 0, 0, 0, err
	}

	for rows.Next() {	
		err = rows.Scan(&symbol, &price, &price_time)
		if err != nil {
			return 0, 0, 0, 0, 0, err
		}

		if symbol == "OBSR" {
			obsr_price = price
			obsr_time = price_time
		} else if symbol == "KLAY" {
			klay_price = price
			klay_time = price_time
		}
	}
	if obsr_time == 0 || klay_time == 0 {
		return 0, 0, 0, 0, 0, errors.New("OBSR 또는 KLAY 가격이 존재하지 않습니다")
	}

	txfee := (klay_price * klay_txfee) / obsr_price
	return math.Round(txfee), obsr_price, obsr_time, klay_price, klay_time, nil
}
*/
func GetTxFee(rds redis.Conn, coin string, fee float64) (map[string]interface{}, error) {

	rkey := global.Config.Service.Name + ":CoinPrice"
	rvalue, err := redis.String(rds.Do("GET", rkey))
	if err != nil { return nil, err }

	mapPrice := make([]interface{}, 0)
	if err = json.Unmarshal([]byte(rvalue), &mapPrice); err != nil { return nil, err }

	res := make(map[string]interface{})
	if strings.EqualFold(coin, "KRW") {

		var obsr_price, obsr_time float64

		for _, price := range mapPrice {
			if strings.EqualFold(price.(map[string]interface{})["symbol"].(string), "OBSR") {
				obsr_price = price.(map[string]interface{})["price"].(float64)
				obsr_time = price.(map[string]interface{})["time"].(float64)
			}
		}

		if obsr_time == 0 {
			return nil, errors.New("OBSR 가격이 존재하지 않습니다")
		}

		res["obsr_price"] = obsr_price
		res["obsr_time"] = obsr_time
		res["txfee"] = RoundFloat64(fee / obsr_price, global.OBSR_PDesz)

	} else if strings.EqualFold(coin, "KLAY") {

		var klay_price, klay_time float64
		var obsr_price, obsr_time float64

		for _, price := range mapPrice {
			if strings.EqualFold(price.(map[string]interface{})["symbol"].(string), "KLAY") {
				klay_price = price.(map[string]interface{})["price"].(float64)
				klay_time = price.(map[string]interface{})["time"].(float64)
			} else if strings.EqualFold(price.(map[string]interface{})["symbol"].(string), "OBSR") {
				obsr_price = price.(map[string]interface{})["price"].(float64)
				obsr_time = price.(map[string]interface{})["time"].(float64)
			}
		}

		if obsr_time == 0 || klay_time == 0 {
			return nil, errors.New("OBSR 또는 KLAY 가격이 존재하지 않습니다")
		}

		res["obsr_price"] = obsr_price
		res["obsr_time"] = obsr_time
		res["klay_price"] = klay_price
		res["klay_time"] = klay_time
		res["txfee"] = RoundFloat64((klay_price * fee) / obsr_price, global.OBSR_PDesz)

	} else {

		res["txfee"] = fee

	}

	return res, nil
}

func GetCoinPrice(db *sql.DB, coin string) {

}

func CheckForbiddenWord(db *sql.DB, ftype string, word string) (bool, error) {

	var count int64

	uWord := strings.ToUpper(word)
	query := "SELECT count(IDX) FROM FORBIDDEN_WORD WHERE (WORD_TYPE = 'S' AND UPPER(WORD) = '" + uWord + "') OR  (WORD_TYPE = 'I' AND '" + uWord + "' LIKE '%'||UPPER(WORD)||'%')"
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return false, err
	}

	if ( count > 0 ) { return true, nil }

	return false, nil
}

//func SendCode_Phone(ncode string, phone string, code string) {
//	global.FLog.Println("SendCode_Phone", ncode, phone, code)
//}

//func SendCode_Email(email string, code string) {
//	global.FLog.Println("SendCode_Email", email, code)
//}
