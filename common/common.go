package common

import (
	"time"
	"errors"
	"strconv"
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

func SendCode_Phone(ncode string, phone string, code string) {
	global.FLog.Println("SendCode_Phone", ncode, phone, code)
}

func SendCode_Email(email string, code string) {
	global.FLog.Println("SendCode_Email", email, code)
}
