package common

import (
	"errors"
	"strconv"
    "io/ioutil"
    "net/http"
	"net/url"
	"reflect"

	"encoding/json"

	"photoApp-server/global"
)

func SMSApi_Send(ncode string, phone string, templateId string, code string) (map[string]interface {}, error) {
	
	if ncode != "82" {
		return nil, errors.New("Only Korea!!")
	}
	
	var msg string
	/*if templateId == "Login" {
		msg = "Login Code [" + code + "]"
	} else if templateId == "Join" {
		msg = "Join Code [" + code + "]"
	} else if templateId == "Joinout" {
		msg = "Joinout Code [" + code + "]"
	} else if templateId == "SearchUser" {
		msg = "SearchUser Code [" + code + "]"
	} else if templateId == "UpdateUser" {
		msg = "UpdateUser Code [" + code + "]"
	} else {
		msg = "Code [" + code + "]"
	}*/
	msg = "OBSERVER [" + code + "] Please enter the authentication code."
	
	return _RequestToSMSApi(phone, msg)
}

func _RequestToSMSApi(receiver string, msg string) (map[string]interface {}, error) {

	global.FLog.Println("_RequestToSMSApi", receiver, msg)

	// 요청 Parameter를 만든다
	postValues := url.Values{
		"key": []string{global.Config.APIs.SMSKey},
		"user_id": []string{global.Config.APIs.SMSUser},
		"sender": []string{global.Config.APIs.SMSSender},
		"receiver": []string{receiver},
		"msg": []string{msg},
	}

	// Post를 요청한다
	resp, err := http.PostForm(global.Config.APIs.SMSApi, postValues)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
	
	// 에러체크
	if resp.StatusCode != http.StatusOK {
		global.FLog.Println("Error", resp)
		return nil, errors.New("Response Code Error [" + strconv.Itoa(resp.StatusCode) + "]")
	}
 
    // 결과 출력
    bytes, _ := ioutil.ReadAll(resp.Body)
	resData := make(map[string]interface{})	
	if err = json.Unmarshal(bytes, &resData); err != nil {
		return nil, err
	}
	global.FLog.Println("Recv", resData)

	var result_code int64
	if reflect.TypeOf(resData["result_code"]).Kind() == reflect.String {
		result_code = GetInt64FromString(resData["result_code"].(string))
	} else {
		result_code = (int64)(resData["result_code"].(float64))
	}

	if result_code < 0 {
		return nil, errors.New(resData["message"].(string))
	}

	return resData, nil
}
