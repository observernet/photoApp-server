package common

import (
	"errors"
	"strconv"
    "io/ioutil"
    "net/http"
	"net/url"

	"encoding/base64"
	"encoding/json"

	"photoApp-server/global"
)

func WSApi_GetUserInfo(email string) (map[string]interface {}, error) {
	return _RequestToWSApi("GetUserInfo", email)
}

func WSApi_GetWSData(serial string) (map[string]interface {}, error) {
	return _RequestToWSApi("GetWSData", serial)
}

func WSApi_UpdateWSRewordStatus(active string, serial string) (map[string]interface {}, error) {
	return _RequestToWSApi("UpdateWSRewordStatus", active, serial)
}

func _RequestToWSApi(trid string, params ...string) (map[string]interface {}, error) {

	// 요청 URL을 만든다
	reqUri := global.Config.APIs.WSApi + "/" + trid
	for _, param := range params {
		reqUri = reqUri + "/" + url.QueryEscape(param)
	}

	// 요청 객체를 생성한다
	req, err := http.NewRequest("GET", reqUri, nil)
    if err != nil {
        return nil, err
    }
 
    //필요시 헤더 추가 가능
	authkey := global.Config.APIs.WSKey + ":" + global.Config.APIs.WSSecret
	bearer := "Bearer " + base64.URLEncoding.EncodeToString([]byte(authkey))
	req.Header.Add("Authorization", bearer)

    // Client객체에서 Request 실행
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

	// 에러체크
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("Response Code Error [" + strconv.Itoa(resp.StatusCode) + "]")
	}
 
    // 결과 출력
    bytes, _ := ioutil.ReadAll(resp.Body)
	resData := make(map[string]interface{})	
	if err = json.Unmarshal(bytes, &resData); err != nil {
		return nil, err
	}

	if resData["code"].(float64) != 0 {
		return nil, errors.New(resData["msg"].(string))
	}

	return resData, nil
}
