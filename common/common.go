package common

import (
	"time"
	"errors"
	"math/rand"

	"photoApp-server/global"

	"database/sql"
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
			result[cols[i]] = item.(*interface{})
		}
		results = append(results, result)

		count = count + 1
		if limit > 0 && count >= limit {
			break
		}
	}

	return results, nil
}

func SendCode_Phone(ncode string, phone string, code string) {
	global.FLog.Println("SendCode_Phone", ncode, phone, code)
}

func SendCode_Email(email string, code string) {
	global.FLog.Println("SendCode_Email", email, code)
}
