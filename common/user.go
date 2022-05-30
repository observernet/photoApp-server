package common

import (
	"time"
	"errors"
	"encoding/json"

	"photoApp-server/global"

	"database/sql"
	"github.com/gomodule/redigo/redis"
)

func User_GetInfo(rds redis.Conn, userkey string) (map[string]interface{}, error) {

	// Redis에서 사용자 정보를 가져온다
	rkey := global.Config.Service.Name + ":UserInfo:" + userkey
	rvalue, err := redis.String(rds.Do("GET", rkey))
	if err != nil { return nil, err }

	// Map 데이타로 변환한다
	mapUser := make(map[string]interface{})
	if err = json.Unmarshal([]byte(rvalue), &mapUser); err != nil { return nil, err }

	return mapUser, nil
}

func User_Login(db *sql.DB, rds redis.Conn, userkey string) (string, error) {

	var err error
	var stmt *sql.Stmt
	var rows *sql.Rows
	var query string
	var mapUser, mapWallet []map[string]interface{}
	curtime := time.Now().UnixNano() / 1000000

	// 사용자 정보를 가져온다
	query = "SELECT USER_KEY, NCODE, PHONE, NVL(EMAIL, ' ') EMAIL, NVL(NAME, ' ') NAME, NVL(PHOTO, ' ') PHOTO, PROMOTION, USER_LEVEL, STATUS, NVL(ABUSE_REASON, ' ') ABUSE_REASON, LABEL_COUNT, LAST_SNAP_TIME FROM USER_INFO WHERE USER_KEY = :1"
	if stmt, err = db.Prepare(query); err != nil { return "", err }
	if rows, err = stmt.Query(userkey); err != nil { return "", err }
	if mapUser, err = GetRowsResult(rows, 1); err != nil { return "", err }
	if len(mapUser) != 1 { return "", errors.New("Not Found User Info") }
	stmt.Close()

	// 지갑 정보를 가져온다
	query = "SELECT ADDRESS, NAME, WALLET_TYPE, CERT_INFO FROM WALLET_INFO WHERE USER_KEY = :1"
	if stmt, err = db.Prepare(query); err != nil { return "", err }
	if rows, err = stmt.Query(userkey); err != nil { return "", err }
	if mapWallet, err = GetRowsResult(rows, 0); err != nil { return "", err }
	defer stmt.Close()

	// redis에 올릴 정보를 생성한다
	mapRedis := make(map[string]interface{})
	mapRedis["info"] = mapUser[0]
	mapRedis["wallet"] = mapWallet

	// Redis에 데이타를 올린다
	jsonStr, _ := json.Marshal(mapRedis)
	rkey := global.Config.Service.Name + ":UserInfo:" + userkey
	if _, err = rds.Do("SET", rkey, jsonStr); err != nil { return "", err }

	// runtime 데이타를 올린다
	mapRedis = map[string]interface{} {"logintime": curtime}

	// Redis에 데이타를 올린다
	jsonStr, _ = json.Marshal(mapRedis)
	rkey = global.Config.Service.Name + ":UserRun:" + userkey
	if _, err = rds.Do("SET", rkey, jsonStr); err != nil { return "", err }

	return rkey, nil
}

func User_Logout(rds redis.Conn, userkey string) error {

	// Redis에서 사용자 정보를 삭제한다
	rkey := global.Config.Service.Name + ":UserInfo:" + userkey
	if _, err := rds.Do("DEL", rkey); err != nil { return err }

	return nil
}