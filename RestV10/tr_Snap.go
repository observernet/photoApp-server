package RestV10

import (
	"fmt"
	"time"
	"strconv"

	//"encoding/json"
	
	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

// ReqData - loginkey: 로그인키
//           lat: 위도
//           lng: 경도
//           alt: 고도
//           bear: 지자기
//           pre: 기압
//           device: 디바이스정보
//           snap: 촬영정보
// ResData - upload_url: 사진업로드 URL
//         - snapkey: 스냅키
func TR_Snap(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	userkey := reqData["key"].(string)
	reqBody := reqData["body"].(map[string]interface{})

	// check input
	if reqBody["loginkey"] == nil { return 9003 }
	if reqBody["lat"] == nil || reqBody["lng"] == nil || reqBody["alt"] == nil || reqBody["bear"] == nil || reqBody["pre"] == nil { return 9003 }
	curtime := time.Now().UnixNano() / 1000000

	//////////////////////////////////////////
	// 여기서 변수 범위값을 체크하자
	//////////////////////////////////////////

	var err error

	// 유저 정보를 가져온다
	var mapUser map[string]interface{}
	if mapUser, err = common.User_GetInfo(rds, userkey); err != nil {
		if err == redis.ErrNil {
			return 8015
		} else {
			global.FLog.Println(err)
			return 9901
		}
	}

	// 관리자 정의변수를 가져온다
	var adminVar global.AdminConfig
	if adminVar, err = common.GetAdminVar(rds); err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 계정 상태 및 로그인 정보를 체크한다
	if mapUser["info"].(map[string]interface{})["STATUS"].(string) != "V" { return 8013 }
	if mapUser["login"].(map[string]interface{})["loginkey"].(string) != reqBody["loginkey"].(string) { return 8014 }

	// 이전 스냅타임을 체크한다
	if (curtime - (int64)(mapUser["stat"].(map[string]interface{})["LAST_SNAP_TIME"].(float64))) < (adminVar.Snap.Interval * 1000) {
		return 8101
	}

	// 지역 체크키를 생성한다
	check_lat := reqBody["lat"].(float64) - ( (float64)((int64)(reqBody["lat"].(float64) * 100000000.0) % (int64)(adminVar.Snap.CheckRange * 100000000.0)) / 100000000.0)
	check_lng := reqBody["lng"].(float64) - ( (float64)((int64)(reqBody["lng"].(float64) * 100000000.0) % (int64)(adminVar.Snap.CheckRange * 100000000.0)) / 100000000.0)
	check_key := fmt.Sprintf("%.5f:%.5f", check_lat, check_lng)
	
	// 지역 최종 스냅시간을 가져온다
	var checkLastSnapTime int64
	rCheckkey := global.Config.Service.Name + ":SnapTime"
	rvalue, err := redis.String(rds.Do("HGET", rCheckkey, check_key))
	if err != nil {
		if err == redis.ErrNil {
			checkLastSnapTime = 0
		} else {
			global.FLog.Println(err)
			return 9901
		}
	} else {
		checkLastSnapTime, _ = strconv.ParseInt(rvalue, 10, 64)
	}

	// 지역 스냅타임을 체크한다
	if (curtime - checkLastSnapTime) < (adminVar.Snap.CheckTime * 1000) {
		return 8102
	}

	// 미세먼지값을 구해온다
	pm10 := 1.0
	pm25 := 2.5

	//////////////////////////////////////////
	// 스냅 등록 처리한다
	var tx *sql.Tx
	var snapIdx int64
	snapDate := time.Now().Format("20060102")

	// 트랜잭션 시작
	tx, err = db.Begin()
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	defer tx.Rollback()

	// 스냅키를 가져온다
	query := "SELECT NVL(MAX(SNAP_IDX), 0) + 1 FROM SNAP WHERE SNAP_DATE = " + snapDate;
	err = tx.QueryRow(query).Scan(&snapIdx)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 스냅정보를 기록한다
	_, err = tx.Exec("INSERT INTO SNAP " +
					 " (SNAP_DATE, SNAP_IDX, SNAP_TIME, USER_KEY, LATD, LNGD, ALTD, BEAR, PRE, PM10, PM25, UPLOAD_STATUS, USER_IP, UPDATE_TIME) " +
					 " VALUES " +
					 " (:1, :2, sysdate, :3, :4, :5, :6, :7, :8, :9, :10, 'A', :11, sysdate) ",
					 snapDate, snapIdx, userkey,
					 reqBody["lat"].(float64), reqBody["lng"].(float64), reqBody["alt"].(float64), reqBody["bear"].(float64), reqBody["pre"].(float64),
					 pm10, pm25, c.ClientIP())
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 사용자 스냅 시간을 기록한다
	_, err = tx.Exec("UPDATE USER_INFO SET LAST_SNAP_TIME = :1 WHERE USER_KEY = :2 ", curtime, userkey)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 트랜잭션 종료
	err = tx.Commit()
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 스냅 등록 처리한다
	//////////////////////////////////////////

	// 캐쉬를 기록한다 기록한다
	common.User_UpdateStat(db, rds, userkey)
	_, err = rds.Do("HSET", rCheckkey, check_key, strconv.FormatInt(curtime, 10))

	// 응답값을 세팅한다
	resBody["upload_url"] = "https://photoapp.obsr-app.org/Snap/Upload"
	resBody["snapkey"] = common.CompressSnapData(fmt.Sprintf("%s%06d", snapDate, snapIdx), reqBody["loginkey"].(string), userkey)

	return 0
}
