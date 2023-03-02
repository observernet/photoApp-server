package common

import (
	"fmt"
	"net"
	"errors"
	"strconv"
	//"reflect"
	//"encoding/json"

	"photoApp-server/global"
)

const _ADDR_LENGTH_SIZE			int = 4
const _ADDR_TYPE_GEOCODE 		string = "C"


func ADDR_GetAddress(lng float64, lat float64) (string, error) {
	return _RequestToAddrConn(_ADDR_TYPE_GEOCODE, fmt.Sprintf("%.6f:%.6f", lng, lat))
}

func _RequestToAddrConn(reqtype string, sendData string) (string, error) {

	var err error
	var hBuff, bBuff []byte

	// 전송할 데이타를 생성한다
	sendBuff := fmt.Sprintf("%0*d%s%s", _ADDR_LENGTH_SIZE, len(sendData) + 1, reqtype, sendData)

	// 서버에 접속한다
	client, err := _ConnectAddress()
	if err != nil {
		return "", nil
	}
	defer client.Close()

	// 데이타를 전송한다
	_, err = client.Write([]byte(sendBuff))
	if err != nil { return "", nil }

	// 헤더를 수신한다
	hBuff = make([]byte, _ADDR_LENGTH_SIZE)
	_, err = client.Read(hBuff)
	if err != nil { return "", err }

	// 나머지 데이타를 수신한다
	length, _ := strconv.Atoi(string(hBuff))
	if length > 0 {
		bBuff = make([]byte, length)
		_, err = client.Read(bBuff)
		if err != nil {
			return "", err
		}
	}

	rcvString := string(bBuff)
	if len(rcvString) <= 1 {
		return "", errors.New("No Return Address")
	}

	return rcvString[1:], nil
}

func _ConnectAddress() (net.Conn, error) {

	// 서버에 연결한다
	conn, err := net.Dial("tcp", global.Config.Connector.AddressHost)
	if err != nil { return nil, err }

	return conn, nil
}
