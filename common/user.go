package common

import (
	"time"
	"context"
	"errors"
	//"strconv"
	"encoding/json"

	"photoApp-server/global"

	"database/sql"
	"github.com/gomodule/redigo/redis"
)

func User_GetInfo(rds redis.Conn, userkey string, hashkey ...string) (map[string]interface{}, error) {

	if len(hashkey) == 0 {
		hashkey = []string{"info", "wallet", "login", "stat"}
	}

	mapUser := make(map[string]interface{})	
	rkey := global.Config.Service.Name + ":UserInfo:" + userkey

	for _, hkey := range hashkey {

		rvalue, err := redis.String(rds.Do("HGET", rkey, hkey))
		if err != nil { return nil, err }

		if hkey == "wallet" {
			mapData := make([]map[string]interface{}, 0)
			if err = json.Unmarshal([]byte(rvalue), &mapData); err != nil {
				return nil, err
			}
			mapUser[hkey] = mapData
		} else {
			mapData := make(map[string]interface{})
			if err = json.Unmarshal([]byte(rvalue), &mapData); err != nil {
				return nil, err
			}
			if hkey == "info" {
				if mapData["MAIN_LANG"] == nil {
					mapData["MAIN_LANG"] = "K"
				}
			}
			mapUser[hkey] = mapData
		}
	}
	
	return mapUser, nil
}

func User_Login(ctx context.Context, db *sql.DB, rds redis.Conn, userkey string) (string, error) {

	var err error
	curtime := time.Now().UnixNano() / 1000000

	// 사용자 정보를 메모리에 올린다
	if err = User_UpdateInfo(ctx, db, rds, userkey); err != nil {
		return "", err
	}

	// 지갑 정보를 메모리에 올린다
	if err = User_UpdateWallet(ctx, db, rds, userkey); err != nil {
		return "", err
	}

	// 통계 정보를 메모리에 올린다
	if err = User_UpdateStat(ctx, db, rds, userkey); err != nil {
		return "", err
	}

	// 로그인키를 생성한다
	loginKey := GetCodeKey(32)

	// Redis에 데이타를 올린다
	mapRedis := map[string]interface{} {"loginkey": loginKey, "logintime": curtime}

	jsonStr, _ := json.Marshal(mapRedis)
	rkey := global.Config.Service.Name + ":UserInfo:" + userkey
	if _, err = rds.Do("HSET", rkey, "login", jsonStr); err != nil { return "", err }

	return loginKey, nil
}

func User_Logout(rds redis.Conn, userkey string) error {

	// Redis에서 사용자 정보를 삭제한다
	rkey := global.Config.Service.Name + ":UserInfo:" + userkey
	_, err := rds.Do("DEL", rkey)
	if err != nil { return err }

	return nil
}

func User_UpdateInfo(ctx context.Context, db *sql.DB, rds redis.Conn, userkey string) error {

	var err error
	//var stmt *sql.Stmt
	var rows *sql.Rows
	var mapUserInfo []map[string]interface{}

	// 사용자 정보를 가져온다
	//query := "SELECT USER_KEY, NCODE, PHONE, NVL(EMAIL, ' ') EMAIL, NVL(NAME, ' ') NAME, NVL(PHOTO, ' ') PHOTO, PROMOTION, USER_LEVEL, STATUS, NVL(ABUSE_REASON, ' ') ABUSE_REASON FROM USER_INFO WHERE USER_KEY = :1"
	//if stmt, err = db.PrepareContext(ctx, query); err != nil { return err }
	//if rows, err = stmt.Query(userkey); err != nil { return err }
	//if mapUserInfo, err = GetRowsResult(rows, 1); err != nil { return err }
	//if len(mapUserInfo) != 1 { return errors.New("Not Found User Info") }
	//stmt.Close()

	query := "SELECT USER_KEY, NCODE, PHONE, NVL(EMAIL, ' ') EMAIL, NVL(NAME, ' ') NAME, NVL(PHOTO, ' ') PHOTO, PROMOTION, USER_LEVEL, STATUS, NVL(ABUSE_REASON, ' ') ABUSE_REASON, NVL(MAIN_LANG, 'K') MAIN_LANG FROM USER_INFO WHERE USER_KEY = '" + userkey + "'"
	if rows, err = db.QueryContext(ctx, query); err != nil { return err }
	if mapUserInfo, err = GetRowsResult(rows, 1); err != nil { return err }
	if len(mapUserInfo) != 1 { return errors.New("Not Found User Info") }

	// Redis에 데이타를 올린다
	jsonStr, _ := json.Marshal(mapUserInfo[0])
	rkey := global.Config.Service.Name + ":UserInfo:" + userkey
	if _, err = rds.Do("HSET", rkey, "info", jsonStr); err != nil { return err }

	return nil
}

func User_UpdateWallet(ctx context.Context, db *sql.DB, rds redis.Conn, userkey string) error {

	var err error
	//var stmt *sql.Stmt
	var rows *sql.Rows
	var mapWallet []map[string]interface{}

	// 지갑 정보를 가져온다
	//query := "SELECT ADDRESS, NAME, WALLET_TYPE, CERT_INFO FROM WALLET_INFO WHERE USER_KEY = :1 and IS_USE = 'Y'"
	//if stmt, err = db.PrepareContext(ctx, query); err != nil { return err }
	//if rows, err = stmt.Query(userkey); err != nil { return err }
	//if mapWallet, err = GetRowsResult(rows, 0); err != nil { return err }
	//defer stmt.Close()

	query := "SELECT ADDRESS, NAME, WALLET_TYPE, CERT_INFO FROM WALLET_INFO WHERE USER_KEY = '" + userkey + "' and IS_USE = 'Y'"
	if rows, err = db.QueryContext(ctx, query); err != nil { return err }
	if mapWallet, err = GetRowsResult(rows, 0); err != nil { return err }

	// Redis에 데이타를 올린다
	jsonStr, _ := json.Marshal(mapWallet)
	rkey := global.Config.Service.Name + ":UserInfo:" + userkey
	if _, err = rds.Do("HSET", rkey, "wallet", jsonStr); err != nil { return err }

	return nil
}

func User_UpdateStat(ctx context.Context, db *sql.DB, rds redis.Conn, userkey string) error {

	var err error
	var rows *sql.Rows
	var label_count, last_snap_time, today_snap_count, today_label_count, today_label_etc_count int64
	var obsp float64 

	// 오늘날짜를 가져온다
	strToday := time.Now().Format("20060102")

	//global.FLog.Println("User_UpdateStat", strToday)

	// 현재 OBSP를 가져온다
	obsp, err = GetUserOBSP(ctx, db, userkey)
	if err != nil { return err }

	// 라벨카운트와 마지막스냅 시간을 가져온다
	err = db.QueryRowContext(ctx, "SELECT LABEL_COUNT, LAST_SNAP_TIME FROM USER_INFO WHERE USER_KEY = '" + userkey + "'").Scan(&label_count, &last_snap_time)
	if err != nil { return err }

	// 당일스냅개수를 가져온다
	err = db.QueryRowContext(ctx, "SELECT count(SNAP_IDX) FROM SNAP WHERE SNAP_DATE = " + strToday + " and USER_KEY = '" + userkey + "' and UPLOAD_STATUS = 'V'").Scan(&today_snap_count)
	if err != nil { return err }

	// 당일 라벨개수를 가져온다
	rows, err = db.QueryContext(ctx, "SELECT IS_ETC, count(LABEL_IDX) FROM SNAP_LABEL WHERE USER_KEY = '" + userkey + "' and TO_CHAR(LABEL_TIME, 'RRRRMMDD') = " + strToday + " GROUP BY IS_ETC")
	if err != nil { return err }
	defer rows.Close()

	var is_etc string
	var tcount int64
	for i := 0; rows.Next(); i++ {
		rows.Scan(&is_etc, &tcount)

		today_label_count = today_label_count + tcount
        if is_etc == "Y" { today_label_etc_count = today_label_etc_count + tcount }
	}

	// Redis에 데이타를 올린다
	mapRedis := map[string]interface{} {
		"OBSP": RoundFloat64(obsp, global.OBSR_PDesz),
		"LABEL_COUNT": label_count,
		"LAST_SNAP_TIME": last_snap_time,
		"TODAY_SNAP_COUNT": today_snap_count,
		"TODAY_LABEL_COUNT": today_label_count,
		"TODAY_LABEL_ETC_COUNT": today_label_etc_count }

	jsonStr, _ := json.Marshal(mapRedis)
	rkey := global.Config.Service.Name + ":UserInfo:" + userkey
	if _, err = rds.Do("HSET", rkey, "stat", jsonStr); err != nil { return err }

	return nil
}