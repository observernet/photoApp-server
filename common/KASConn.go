package common

import (
	"fmt"
	"net"
	"errors"
	"strconv"
	"reflect"
	"encoding/json"

	"photoApp-server/global"
)

// KASConn 관련 정의 (https://github.com/observernet/Klaytn-Connector/blob/master/include/KASConn_interface.h 참조)
const _KI_HEADER_SIZE			int = 72
const _KI_BODY_LENGTH			int = 5

const _KI_REQTYPE_LOGIN 		string = "L"
const _KI_REQTYPE_CALL 			string = "C"
const _KI_REQTYPE_TRANSACT 		string = "T"

const _KI_TRID_LOGIN			string = "L"
const _KI_TRID_CREATE_ACCOUNT	string = "C"
const _KI_TRID_BALANCEOF		string = "B"
const _KI_TRID_TRANSFER			string = "T"

type _KI_REQRES_HEADER struct
{
	Trid			string		// 1
	ReqType     	string		// 1
	ServiceName 	string		// 16
	AccountType     string		// 1
	AccountGroup	string		// 16
	UserKey			string		// 32
	BodyLength		int			// _KI_BODY_LENGTH
}


func KAS_CreateAccount(userkey string) (string, error) {
	return _InquiryCallToKASConn(_KI_TRID_CREATE_ACCOUNT, "K", userkey, `{"cert":"` + global.Config.Service.AccountPool + `"}`)
}

func KAS_GetBalanceOf(userkey string, address string, address_type string) (string, error) {
	return _InquiryCallToKASConn(_KI_TRID_BALANCEOF, address_type, userkey, `{"address":"` + address + `"}`)
}

func KAS_Transfer(userkey string, sender string, recipient string, amount string, address_type string, cert_info string) (string, error) {
	return _TransactToKASConn(_KI_TRID_TRANSFER, address_type, userkey, `{"sender":"` + sender + `", "recipient": "` + recipient + `", "amount": "` + amount + `", "cert": "` + cert_info + `"}`)
}

func _InquiryCallToKASConn(trid string, acctype string, userkey string, sendData string) (string, error) {

	var err error

	// 전송할 데이타를 생성한다
	sendBuff := fmt.Sprintf("%s%s%-16s%s%-16s%-32s%0*d%s", trid, _KI_REQTYPE_CALL, global.Config.Service.Name, acctype, global.Config.Service.AccountPool, userkey, _KI_BODY_LENGTH, len(sendData), sendData)

	var header _KI_REQRES_HEADER
	var hBuff, bBuff []byte

	// 서버에 접속한다
	client, err := _ConnectKASInquiry()
	if err != nil {
		return "", nil
	}
	defer client.Close()

	// 데이타를 전송한다
	_, err = client.Write([]byte(sendBuff))
	if err != nil { return "", nil }

	// 헤더를 수신한다
	hBuff = make([]byte, _KI_HEADER_SIZE)
	_, err = client.Read(hBuff)
	if err != nil { return "", err }

	// 나머지 데이타를 수신한다
	_ConvertKASRegResHeader(string(hBuff), &header)
	if header.BodyLength > 0 {
		bBuff = make([]byte, header.BodyLength)
		_, err = client.Read(bBuff)
		if err != nil {
			return "", err
		}
	}

	var res map[string]interface{}
	if err = json.Unmarshal(bBuff, &res); err != nil {
		return "", err
	}

	if res["success"] == nil || res["msg"] == nil {
		return "", errors.New("Incorrect KASConn result")
	}

	if res["success"].(bool) != true {
		return "", errors.New(res["msg"].(string))
	}

	var retVal string
	typeVal := reflect.TypeOf(res["msg"])
	switch typeVal.Kind() { 
		case reflect.Float64:	retVal = fmt.Sprintf("%f", res["msg"].(float64))
		case reflect.String:	retVal = res["msg"].(string)
	}

	return retVal, nil
}

func _TransactToKASConn(trid string, acctype string, userkey string, sendData string) (string, error) {

	var err error

	// 전송할 데이타를 생성한다
	sendBuff := fmt.Sprintf("%s%s%-16s%s%-16s%-32s%0*d%s", trid, _KI_REQTYPE_TRANSACT, global.Config.Service.Name, acctype, global.Config.Service.AccountPool, userkey, _KI_BODY_LENGTH, len(sendData), sendData)

	var header _KI_REQRES_HEADER
	var hBuff, bBuff []byte

	// 서버에 접속한다
	client, err := _ConnectKASTransact()
	if err != nil {
		return "", nil
	}
	defer client.Close()

	// 데이타를 전송한다
	_, err = client.Write([]byte(sendBuff))
	if err != nil {
		return "", nil
	}

	// 헤더를 수신한다
	hBuff = make([]byte, _KI_HEADER_SIZE)
	_, err = client.Read(hBuff)
	if err != nil {
		return "", err
	}

	// 나머지 데이타를 수신한다
	_ConvertKASRegResHeader(string(hBuff), &header)
	if header.BodyLength > 0 {
		bBuff = make([]byte, header.BodyLength)
		_, err = client.Read(bBuff)
		if err != nil { return "", err }
	}

	return string(bBuff), nil
}

func _ConnectKASInquiry() (net.Conn, error) {

	// 로그인 데이타를 생성한다
	loginBuff := fmt.Sprintf("%s%s%-16s%s%-16s%-32s%0*d", _KI_TRID_LOGIN, _KI_REQTYPE_LOGIN, global.Config.Service.Name, " ", global.Config.Service.AccountPool, " ", _KI_BODY_LENGTH, 0)

	// 서버에 연결한다
	conn, err := net.Dial("tcp", global.Config.Connector.KASInquiryHost)
	if err != nil { return nil, err }

	// 로그인 데이타를 전송한다
	_, err = conn.Write([]byte(loginBuff))
	if err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func _ConnectKASTransact() (net.Conn, error) {

	// 로그인 데이타를 생성한다
	loginBuff := fmt.Sprintf("%s%s%-16s%s%-16s%-32s%0*d", _KI_TRID_LOGIN, _KI_REQTYPE_LOGIN, global.Config.Service.Name, " ", global.Config.Service.AccountPool, " ", _KI_BODY_LENGTH, 0)

	// 서버에 연결한다
	conn, err := net.Dial("tcp", global.Config.Connector.KASTransactHost)
	if err != nil { return nil, err }

	// 로그인 데이타를 전송한다
	_, err = conn.Write([]byte(loginBuff))
	if err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func _ConvertKASRegResHeader(stream string, header *_KI_REQRES_HEADER) {

	var offset int = 0

	header.Trid = stream[offset:offset+1]; offset = offset + 1
	header.ReqType = stream[offset:offset+1]; offset = offset + 1
	header.ServiceName = stream[offset:offset+16]; offset = offset + 16
	header.AccountType = stream[offset:offset+1]; offset = offset + 1
	header.AccountGroup = stream[offset:offset+16]; offset = offset + 16
	header.UserKey = stream[offset:offset+32]; offset = offset + 32

	length := stream[offset:offset+_KI_BODY_LENGTH]
	header.BodyLength, _ = strconv.Atoi(length)
}
