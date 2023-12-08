package RestV10

import (
	"time"
	"context"
	
	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

// ReqData - loginkey: 로그인키
// ResData - stat
func TR_AdReword(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	ctx, cancel := context.WithTimeout(c, global.DBContextTimeout * time.Second)
	defer cancel()

	userkey := reqData["key"].(string)
	reqBody := reqData["body"].(map[string]interface{})

	// check input
	if reqBody["loginkey"] == nil { return 9003 }
	curtime := time.Now().UnixNano() / 1000000

	//////////////////////////////////////////
	// 여기서 변수 범위값을 체크하자

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

	// 라벨 갯수를 증가한다
	_, err = db.ExecContext(ctx, "UPDATE USER_INFO SET LABEL_COUNT = LABEL_COUNT + :1 WHERE USER_KEY = :2 ", adminVar.Label.AddAdLabel, userkey)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 캐쉬를 기록한다 기록한다
	common.User_UpdateStat(ctx, db, rds, userkey)

	// 통계정보를 가져온다
	mapStat, _ := common.User_GetInfo(rds, userkey)
	remain_snap_time := curtime - (int64)(mapStat["stat"].(map[string]interface{})["LAST_SNAP_TIME"].(float64))
	remain_snap_time = adminVar.Snap.Interval - remain_snap_time / 1000
	if remain_snap_time < 0 { remain_snap_time = 0 }

	// 응답값을 세팅한다
	resBody["adreword"] = adminVar.Label.AddAdLabel
	resBody["stat"] = map[string]interface{} {
		"obsp": common.RoundFloat64(mapUser["stat"].(map[string]interface{})["OBSP"].(float64), global.OBSR_PDesz),
		"labels": mapStat["stat"].(map[string]interface{})["LABEL_COUNT"].(float64),
		"remain_snap_time": remain_snap_time,
		"count": map[string]interface{} {
			"snap": mapStat["stat"].(map[string]interface{})["TODAY_SNAP_COUNT"].(float64),
			"snap_rp": mapStat["stat"].(map[string]interface{})["TODAY_SNAP_COUNT"].(float64) * adminVar.Reword.Snap,
			"label": mapStat["stat"].(map[string]interface{})["TODAY_LABEL_COUNT"].(float64),
			"label_rp": mapStat["stat"].(map[string]interface{})["TODAY_LABEL_COUNT"].(float64) * adminVar.Reword.Label,
			"label_etc": mapStat["stat"].(map[string]interface{})["TODAY_LABEL_ETC_COUNT"].(float64),
			"label_etc_rp": mapStat["stat"].(map[string]interface{})["TODAY_LABEL_ETC_COUNT"].(float64) * adminVar.Reword.LabelEtc}}

	return 0
}
