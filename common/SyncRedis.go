package common

import (
	"fmt"
	"net"
	"strconv"

	"photoApp-server/global"
)

// SyncRedis 관련 정의 (https://github.com/observernet/SynchronizeRedis/blob/master/include/SyncRedis_interface.h 참조)
const _SRI_LENGTH_SIZE			int = 4
const _SRI_DATA_SEPERATOR		string = "|"

const _SRI_TRID_UPDATE			string = "U"
const _SRI_TRID_DELETE			string = "D"

// 소켓 정보
var _SyncRedisConn net.Conn


func SyncRedis_Update(keys ...string) (string, error) {
	
	var keylist string
	for _, key := range keys {
		keylist = keylist + key + _SRI_DATA_SEPERATOR
	}
	keylist = keylist[0:len(keylist)-1]

	return _SendEventToSyncRedis(_SRI_TRID_UPDATE, keylist)
}

func _SendEventToSyncRedis(trid string, sendData string) (string, error) {

	var err error

	// 전송할 데이타를 생성한다
	sendBuff := fmt.Sprintf("%0*d%s%s", _SRI_LENGTH_SIZE, len(sendData) + 1, trid, sendData)

	var sBuff, rBuff []byte

	for i := 0; i < 5; i++ {

		// 연결정보가 없다면 접속한다
		if _SyncRedisConn == nil {
			if err = _ConnectSyncRedis(); err != nil {
				return "", nil
			}
		}

		// 데이타를 전송한다
		_, err = _SyncRedisConn.Write([]byte(sendBuff))
		if err != nil {
			_SyncRedisConn.Close()
			_SyncRedisConn = nil
			continue
		}

		// 헤더를 수신한다
		sBuff = make([]byte, _SRI_LENGTH_SIZE)
		_, err = _SyncRedisConn.Read(sBuff)
		if err != nil {
			return "", err
		}

		// 나머지 데이타를 수신한다
		length, _ := strconv.Atoi(string(sBuff))
		if length > 0 {
			rBuff = make([]byte, length)
			_, err = _SyncRedisConn.Read(rBuff)
			if err != nil { return "", err }
		}

		break
	}
	result := string(rBuff)
	result = result[1:len(result)]

	return result, nil
}

func _ConnectSyncRedis() (error) {

	var err error

	// 서버에 연결한다
	_SyncRedisConn, err = net.Dial("tcp", global.Config.Connector.SyncRedisHost)
	if err != nil { return err }

	return nil
}
