package common

import (
	"fmt"
	"time"
	"strings"
	"errors"
	"strconv"
    "io/ioutil"
    "net/http"
	"net/url"
	"reflect"

	"encoding/base64"
	"encoding/json"

	"photoApp-server/global"
)

func SMSApi_Send(ncode string, phone string, templateId string, code string) (map[string]interface {}, error) {
	
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

	if ncode == "82" {
		return _RequestToSMSApi(phone, msg)
	} else {
		return _RequestToSMSApiGabia(ncode + phone, msg)
	}
	
	return nil, errors.New("Can not reach!")
}

func _RequestToSMSApi(receiver string, msg string) (map[string]interface{}, error) {

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

func _RequestToSMSApiGabia(receiver string, msg string) (map[string]interface{}, error) {

	global.FLog.Println("_RequestToSMSApiGabia", receiver, msg)

	accessToken, err := _GetGabiaAccessToken()
	if err != nil { return nil, err }

	target := "https://sms.gabia.com/api/send/sms"
	method := "POST"

	dt := time.Now()
	uniqueKey := "GO" + dt.Format("20060102150405") + fmt.Sprintf("%02d", dt.Nanosecond() / 1000000)

	payload := strings.NewReader("phone=" + receiver +
							     "&callback=" + global.Config.APIs.GabiaSMSSender +
							     "&message=" + url.QueryEscape(msg) +
								 "&is_foreign=Y" + 
							     "&refkey=" + uniqueKey)
	
	//fmt.Println(payload)

	client := &http.Client {
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequest(method, target, payload)
	if err != nil { return nil, err }

	platKey := global.Config.APIs.GabiaSMSUser + ":" + accessToken
	encKey := base64.StdEncoding.EncodeToString([]byte(platKey))

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", "Basic " + encKey)

	res, err := client.Do(req)
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	fmt.Println(string(body))

	resData := make(map[string]interface{})	
	if err = json.Unmarshal(body, &resData); err != nil {
		return nil, err
	}

	if resData["message"] != nil && !strings.EqualFold(resData["message"].(string), "success") {
		return nil, errors.New(resData["message"].(string))
	}

	return resData, nil
}

func _GetGabiaAccessToken() (string, error) {

	target := "https://sms.gabia.com/oauth/token"
	method := "POST"

	payload := strings.NewReader("grant_type=client_credentials")

	client := &http.Client {
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequest(method, target, payload)
	if err != nil { return "", err }

	platKey := global.Config.APIs.GabiaSMSUser + ":" + global.Config.APIs.GabiaSMSKey
	encKey := base64.StdEncoding.EncodeToString([]byte(platKey))

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", "Basic " + encKey)

	res, err := client.Do(req)
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	resData := make(map[string]interface{})	
	if err = json.Unmarshal(body, &resData); err != nil {
		return "", err
	}

	if resData["message"] != nil {
		return "", errors.New(resData["message"].(string))
	}

	if resData["access_token"] == nil {
		return "", errors.New("Invalid Access Token")
	}

	return resData["access_token"].(string), nil
}
