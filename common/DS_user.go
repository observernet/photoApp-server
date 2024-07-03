package common

import (
	"fmt"
	"time"
	"context"
	//"errors"
	//"strconv"
	"encoding/json"

	//"photoApp-server/global"

	"database/sql"
	"github.com/gomodule/redigo/redis"
)

func DSUser_Login(rds redis.Conn, userkey string) (string, error) {

	var err error

	// 로그인키를 생성한다
	curtime := time.Now().UnixNano() / 1000000
	loginKey := GetCodeKey(32)

	// Redis에 데이타를 올린다
	mapRedis := map[string]interface{} {"loginkey": loginKey, "logintime": curtime}

	jsonStr, _ := json.Marshal(mapRedis)
	if _, err = rds.Do("HSET", "DataStore:UserInfo", userkey, jsonStr); err != nil { return "", err }

	return loginKey, nil
}

func DSUser_Logout(rds redis.Conn, userkey string) error {

	// Redis에서 사용자 정보를 삭제한다
	_, err := rds.Do("HDEL", "DataStore:UserInfo", userkey)
	if err != nil { return err }

	return nil
}

func DSUser_GetInfo(ctx context.Context, db *sql.DB, rds redis.Conn, userkey string) (map[string]interface{}, error) {

	userInfo := make(map[string]interface{})

	// Fetch User Info (From Redis)
	rvalue, err := redis.String(rds.Do("HGET", "DataStore:UserInfo", userkey))
	if err != nil {
		if err == redis.ErrNil {
			rvalue = ""
		} else {
			fmt.Println(err)
			return nil, err
		}
	}
	mapLogin := map[string]interface{} {"loginkey": "", "logintime": float64(0)}
	if len(rvalue) > 0 {
		if err = json.Unmarshal([]byte(rvalue), &mapLogin); err != nil {
			fmt.Println(err)
			return nil, err
		}
	}

	// Fetch User Info (From DB)
	var ncode, phone, email, name, promotion, lang string
	var level int64
	query := "SELECT NCODE, PHONE, NVL(EMAIL, ' ') EMAIL, NVL(NAME, ' ') NAME, PROMOTION, USER_LEVEL, NVL(MAIN_LANG, 'K') MAIN_LANG FROM USER_INFO WHERE USER_KEY = '" + userkey + "'"
	err = db.QueryRowContext(ctx, query).Scan(&ncode, &phone, &email, &name, &promotion, &level, &lang)
	if err != nil { return nil, err }

	// Fetch Wallet Info (From DB)
	query = "SELECT ADDRESS, NAME, WALLET_TYPE FROM WALLET_INFO WHERE USER_KEY = '" + userkey + "' and IS_USE = 'Y'"
	rows, err := db.QueryContext(ctx, query)
	if err != nil { return nil, err }
	defer rows.Close()

	var address, wallet_type string
	wallets := make([]map[string]interface{}, 0)
	for rows.Next() {	
		err = rows.Scan(&address, &name, &wallet_type)
		if err != nil { return nil, err }
		
		wallets = append(wallets, map[string]interface{} {"address": address, "type": wallet_type, "name": name})
	}

	// Set UserInfo
	userInfo["userkey"] = userkey
	userInfo["loginkey"] = mapLogin["loginkey"].(string)
	userInfo["logintime"] = mapLogin["logintime"].(float64)
	userInfo["info"] = map[string]interface{} {
		"ncode": ncode,
		"phone": phone,
		"email": email,
		"name":  name,
		"photo": "https://photoapp.obsr-app.org/Image/View/profile/" + userkey,
		"level": level,
		"promotion": promotion,
		"lang": lang,
	}
	userInfo["wallet"] = wallets

	return userInfo, nil
}

func DSUser_GetLoginInfo(rds redis.Conn, userkey string) (map[string]interface{}, error) {

	// Fetch User Info (From Redis)
	rvalue, err := redis.String(rds.Do("HGET", "DataStore:UserInfo", userkey))
	if err != nil {
		if err == redis.ErrNil {
			rvalue = ""
		} else {
			fmt.Println(err)
			return nil, err
		}
	}
	mapLogin := map[string]interface{} {"loginkey": "", "logintime": float64(0)}
	if len(rvalue) > 0 {
		if err = json.Unmarshal([]byte(rvalue), &mapLogin); err != nil {
			fmt.Println(err)
			return nil, err
		}
	}

	return mapLogin, nil
}