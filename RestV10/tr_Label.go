package RestV10

import (
	"fmt"
	"time"
	"context"
	"encoding/json"
	
	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

// ReqData - loginkey: 로그인키
//         - snapkey: 스냅키
//         - status: 사진상태 (0:정상, 1:신고)
//         - sky: 하늘상태 (N:없음, Y:있음, F:가득)
//         - rain: 강수상태 (N:없음, Y:비, S:눈)
//         - wcondi: 자연현상 (N:없음, F:안개, H:우박, R:무지개, T:번개)
//         - calamity: 재난현상 (N:없음, R:폭우, S:폭설, F:산불, L:산사태)
//         - accuse: 신고구분 (V:날씨확인불가, F:직접촬영한날씨사진아님, A:광고, S:음란폭력혐오, P:초상권침해)
//         - is_etc: 기타라벨링여부 (true, false)
// ResData - snapkey: 스냅키
//         - labelkey: 라벨키
//         - rp: 오늘 RP {snap: }
func TR_Label(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	ctx, cancel := context.WithTimeout(c, global.DBContextTimeout * time.Second)
	defer cancel()

	userkey := reqData["key"].(string)
	reqBody := reqData["body"].(map[string]interface{})

	// check input
	if reqBody["loginkey"] == nil || reqBody["snapkey"] == nil { return 9003 }
	if reqBody["status"] == nil || (reqBody["status"].(string) != "0" && reqBody["status"].(string) != "1") { return 9003 }
	if reqBody["status"].(string) == "0" && (reqBody["sky"] == nil || reqBody["rain"] == nil) { return 9003 }
	if reqBody["status"].(string) == "1" && reqBody["accuse"] == nil { return 9003 }
	if reqBody["is_etc"] != nil && reqBody["is_etc"].(bool) == true && (reqBody["wcondi"] == nil || reqBody["calamity"] == nil) { return 9003 }
	curtime := time.Now().UnixNano() / 1000000

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

	// 버전이 세팅되어 있다면
	if reqBody["os"] != nil && reqBody["version"] != nil {
		if reqBody["os"].(string) != "android" && reqBody["os"].(string) != "ios" { return 9003 }
		if len(reqBody["version"].(string)) == 0 { return 9003 }

		// 버전 정보를 가져온다
		rkey := global.Config.Service.Name + ":Version"
		rvalue, err := redis.String(rds.Do("GET", rkey))
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}

		// Map으로 변환한다
		mapVer := make(map[string]interface{})	
		if err = json.Unmarshal([]byte(rvalue), &mapVer); err != nil {
			global.FLog.Println(err)
			return 9901
		}

		min_version := mapVer[reqBody["os"].(string) + "_min"].(string)
		if common.CheckVersion(reqBody["version"].(string), min_version) == false {
			return 9903
		}
	} else {
		return 9903
	}

	// 현재 사용자의 라벨가능 건수를 체크한다
	if mapUser["stat"].(map[string]interface{})["LABEL_COUNT"].(float64) <= 0 {
		return 8103
	}

	// 스냅키를 가져온다
	var snap_date, snap_idx string
	snap_date, snap_idx, err = common.GetSnapKey(reqBody["snapkey"].(string))
	if err != nil {
		global.FLog.Println(err)
		return 8104
	}

	// 해당스냅의 정보를 가져온다
	var row_mylabels int64
	var row_user_key, row_is_show, row_upload_status string
	query := "SELECT USER_KEY, IS_SHOW, UPLOAD_STATUS, (SELECT count(LABEL_IDX) FROM SNAP_LABEL WHERE SNAP_DATE = A.SNAP_DATE and SNAP_IDX = A.SNAP_IDX and USER_KEY = '" + userkey + "') " +
	         "FROM SNAP A " +
			 "WHERE SNAP_DATE = " + snap_date + " and SNAP_IDX = " + snap_idx;
	err = db.QueryRowContext(ctx, query).Scan(&row_user_key, &row_is_show, &row_upload_status, &row_mylabels)
	if err != nil {
		if err == sql.ErrNoRows {
			return 8104
		} else {
			global.FLog.Println(err)
			return 9901
		}
	}

	// 스냅정보를 체크한다
	if row_user_key == userkey { return 8105 }
	if row_is_show != "Y" { return 8106 }
	if row_upload_status != "V" { return 8107 }
	if row_mylabels > 0 { return 8108 }

	// 라벨건수를 가져온다
	var row_labels int64
	query = "SELECT count(LABEL_IDX) FROM SNAP_LABEL WHERE SNAP_DATE = " + snap_date + " and SNAP_IDX = " + snap_idx;
	err = db.QueryRowContext(ctx, query).Scan(&row_labels)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	if row_labels >= adminVar.Label.MaxPerSnap { return 8109 }

	//////////////////////////////////////////
	// 라벨 등록 처리한다
	var tx *sql.Tx
	var labelIdx int64

	// 트랜잭션 시작
	tx, err = db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	defer tx.Rollback()

	// 라벨키를 가져온다
	query = "SELECT NVL(MAX(LABEL_IDX), 0) + 1 FROM SNAP_LABEL WHERE SNAP_DATE = " + snap_date + " and SNAP_IDX = " + snap_idx;
	err = tx.QueryRow(query).Scan(&labelIdx)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 라벨정보를 기록한다
	if reqBody["status"].(string) == "1" {
		_, err = tx.Exec("INSERT INTO SNAP_LABEL " +
						 " (SNAP_DATE, SNAP_IDX, LABEL_IDX, LABEL_TIME, USER_KEY, STATUS, ACCUSE, USER_IP, UPDATE_TIME) " + 
						 " VALUES " +
						 " (:1, :2, :3, sysdate, :4, :5, :6, :7, sysdate) ",
						 snap_date, snap_idx, labelIdx, userkey, reqBody["status"].(string), reqBody["accuse"].(string), c.ClientIP())
	} else {
		if reqBody["is_etc"].(bool) {
			_, err = tx.Exec("INSERT INTO SNAP_LABEL " +
							 " (SNAP_DATE, SNAP_IDX, LABEL_IDX, LABEL_TIME, USER_KEY, STATUS, SKY, RAIN, WCONDI, CALAMITY, IS_ETC, USER_IP, UPDATE_TIME) " + 
							 " VALUES " +
							 " (:1, :2, :3, sysdate, :4, :5, :6, :7, :8, :9, 'Y', :10, sysdate) ",
							 snap_date, snap_idx, labelIdx, userkey, reqBody["status"].(string), reqBody["sky"].(string), reqBody["rain"].(string), reqBody["wcondi"].(string), reqBody["calamity"].(string), c.ClientIP())
		} else {
			_, err = tx.Exec("INSERT INTO SNAP_LABEL " +
							 " (SNAP_DATE, SNAP_IDX, LABEL_IDX, LABEL_TIME, USER_KEY, STATUS, SKY, RAIN, IS_ETC, USER_IP, UPDATE_TIME) " + 
							 " VALUES " +
							 " (:1, :2, :3, sysdate, :4, :5, :6, :7, 'N', :8, sysdate) ",
							 snap_date, snap_idx, labelIdx, userkey, reqBody["status"].(string), reqBody["sky"].(string), reqBody["rain"].(string), c.ClientIP())
		}
	}
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}

	// 신고구분이 음란폭력이면
	if reqBody["status"].(string) == "1" && reqBody["accuse"].(string) == "S" {
		_, err = tx.Exec("UPDATE SNAP SET IS_SHOW = 'N', UPDATE_TIME = sysdate WHERE SNAP_DATE = :1 and SNAP_IDX = :2", snap_date, snap_idx)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}
	}

	// 라벨카운트를 갱신한다
	_, err = tx.Exec("UPDATE USER_INFO SET LABEL_COUNT = :1 WHERE USER_KEY = :2 ", mapUser["stat"].(map[string]interface{})["LABEL_COUNT"].(float64) - 1, userkey)
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

	// 라벨 등록 처리한다
	//////////////////////////////////////////

	// 캐쉬를 기록한다 기록한다
	common.User_UpdateStat(ctx, db, rds, userkey)

	// 통계정보를 가져온다
	mapStat, _ := common.User_GetInfo(rds, userkey)
	remain_snap_time := curtime - (int64)(mapStat["stat"].(map[string]interface{})["LAST_SNAP_TIME"].(float64))
	remain_snap_time = adminVar.Snap.Interval - remain_snap_time / 1000
	if remain_snap_time < 0 { remain_snap_time = 0 }

	// 응답값을 세팅한다
	resBody["labelkey"] = fmt.Sprintf("%s%03d", reqBody["snapkey"].(string), labelIdx)
	//if reqBody["status"].(string) == "1" {
	//	resBody["accuse_rp"] = adminVar.Reword.Label
	//} else {
		resBody["label_rp"] = adminVar.Reword.Label
		if reqBody["is_etc"].(bool) { resBody["label_etc_rp"] = adminVar.Reword.LabelEtc }
	//}
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
