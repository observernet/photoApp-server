package common

import (
	"fmt"
	"net"

	"photoApp-server/global"
)

// MQTTPusher 관련 정의 (https://github.com/observernet/photoApp-pusher/blob/master/include/MQTTPusher_interface.h 참조)
const _MPI_LENGTH_SIZE			int = 4

// 소켓 정보
var _MQTTPusherConn net.Conn


func MQTTPusher_Emit(topic string, message string) (error) {
	
	return _SendEventToMQTTPusher(topic, message)
}

func _SendEventToMQTTPusher(topic string, message string) (error) {

	var err error
	var version string = "v2"

	// 전송할 데이타를 생성한다
	sendBuff := fmt.Sprintf("%0*d%s\t%s\t%s", _MPI_LENGTH_SIZE, len(version) + len(topic) + len(message) + 2, version, topic, message)

	for i := 0; i < 5; i++ {

		// 연결정보가 없다면 접속한다
		if _MQTTPusherConn == nil {
			if err = _ConnectMQTTPusher(); err != nil {
				return err
			}
		}

		// 데이타를 전송한다
		_, err = _MQTTPusherConn.Write([]byte(sendBuff))
		if err != nil {
			_MQTTPusherConn.Close()
			_MQTTPusherConn = nil
			continue
		}

		break
	}

	return nil
}

func _ConnectMQTTPusher() (error) {

	var err error

	// 서버에 연결한다
	_MQTTPusherConn, err = net.Dial("tcp", global.Config.Connector.MQTTPusherHost)
	if err != nil { return err }

	return nil
}
