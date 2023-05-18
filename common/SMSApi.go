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
	"bytes"

	"crypto/hmac"
	"crypto/sha256"
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
	//msg = "OBSERVER [" + code + "] Please enter the authentication code."
	msg = "[OBSERVER] verification: " + code

	if ncode == "82" {
		return _RequestToSMSApi(phone, msg)
	} else if ncode == "62" {
		return _RequestToSMSApiOval(phone, msg)
	} else {
		return _RequestToSMSApiNCloud(ncode, phone, msg)
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

func _RequestToSMSApiNCloud(ncode string, receiver string, msg string) (map[string]interface{}, error) {

	global.FLog.Println("_RequestToSMSApiNCloud", receiver, msg)

	timestamp := fmt.Sprintf("%d", time.Now().UnixNano() / 1000000)

	method := "POST"
	url := "https://sens.apigw.ntruss.com"
	uri := "/sms/v2/services/" + global.Config.APIs.NCloudServiceID + "/messages"
	signature := _GetNCloudSignature(timestamp, method, uri)

	fmt.Println("--->", timestamp, signature)

	message := make([]map[string]interface{}, 0)
	message = append(message, map[string]interface{} {"to": receiver})

	reqBody := make(map[string]interface{})
	reqBody["type"] = "SMS"
	reqBody["contentType"] = "COMM"
	reqBody["countryCode"] = ncode
	reqBody["from"] = global.Config.APIs.NCloudSMSSender
	reqBody["content"] = msg
	reqBody["messages"] = message
	pbytes, _ := json.Marshal(reqBody)
    buff := bytes.NewBuffer(pbytes)

	fmt.Println("--->", string(pbytes))

	client := &http.Client {
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequest(method, url + uri, buff)
	if err != nil { return nil, err }

	req.Header.Add("Content-Type", "application/json; charset=utf-8")
	req.Header.Add("x-ncp-apigw-timestamp", timestamp)
	req.Header.Add("x-ncp-iam-access-key", global.Config.APIs.NCloudAccessKey)
	req.Header.Add("x-ncp-apigw-signature-v2", signature)

	res, err := client.Do(req)
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	fmt.Println("--->", string(body))

	resData := make(map[string]interface{})	
	if err = json.Unmarshal(body, &resData); err != nil {
		return nil, err
	}

	if resData["error"] != nil {
		return nil, errors.New("Header Error")
	}

	if resData["status"] != nil {
		return nil, errors.New(resData["errorMessage"].(string))
	}

	if resData["statusCode"] != nil && !strings.EqualFold(resData["statusCode"].(string), "202") {
		return nil, errors.New(resData["statusCode"].(string) + " " + resData["statusName"].(string))
	}

	return resData, nil
}

func _GetNCloudSignature(timestamp string, method string, uri string) (string) {

	message := method + " " + uri + "\n" + timestamp + "\n" + global.Config.APIs.NCloudAccessKey

	mac := hmac.New(sha256.New, []byte(global.Config.APIs.NCloudSecretKey))
	mac.Write([]byte(message))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return signature
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

func _RequestToSMSApiOval(receiver string, msg string) (map[string]interface{}, error) {

	global.FLog.Println("_RequestToSMSApiOval", receiver, msg)

	method := "POST"
	url := "https://api.ovalsms.com/api/v2/SendSMS"

	reqBody := make(map[string]interface{})
	reqBody["ApiKey"] = global.Config.APIs.OvalAPIKey
	reqBody["ClientId"] = global.Config.APIs.OvalClientId
	reqBody["SenderId"] = global.Config.APIs.OvalSMSSender
	reqBody["Message"] = msg
	reqBody["MobileNumbers"] = "62" + receiver
	pbytes, _ := json.Marshal(reqBody)
    buff := bytes.NewBuffer(pbytes)

	fmt.Println("Request --->", string(pbytes))

	client := &http.Client {
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequest(method, url, buff)
	if err != nil { return nil, err }

	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	fmt.Println("Response --->", string(body))

	resData := make(map[string]interface{})	
	if err = json.Unmarshal(body, &resData); err != nil {
		return nil, err
	}

	if resData["Data"] == nil {
		return nil, errors.New("Response Error")
	}

	if len(resData["Data"].([]interface{})) != 1 {
		return nil, errors.New("Response Error 2")
	}

	result := resData["Data"].([]interface{})[0].(map[string]interface{})

	if result["MessageErrorCode"] != nil && result["MessageErrorCode"].(float64) > 0 {
		return nil, errors.New(result["MessageErrorDescription"].(string))
	}

	if result["MessageId"] == nil || len(result["MessageId"].(string)) == 0 {
		return nil, errors.New("MessageId not Found")
	}

	return resData, nil
}
