package RestV10

import (
	"fmt"
	"strings"

	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

type _PersonaData struct {
	Col			string
	Val			string
}

// ReqData - 
// ResData - 
func TR_Persona(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	userkey := reqData["key"].(string)
	reqBody := reqData["body"].(map[string]interface{})

	// check input
	if reqBody["loginkey"] == nil { return 9003 }

	var err error

	// 관리자변수를 가져온다
	var adminVar global.AdminConfig
	if adminVar, err = common.GetAdminVar(rds); err != nil {
		global.FLog.Println(err)
		return 9901
	}

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

	// 계정 상태 및 로그인 정보를 체크한다
	if mapUser["info"].(map[string]interface{})["STATUS"].(string) != "V" { return 8013 }
	if mapUser["login"].(map[string]interface{})["loginkey"].(string) != reqBody["loginkey"].(string) { return 8014 }

	// 현재 지갑의 수량을 체크한다
	wallet_count := len(mapUser["wallet"].([]map[string]interface{}))
	if wallet_count == 0 { return 8203 }
	if wallet_count > 1 { return 8215 }

	// 현재 지갑의 타입을 체크한다
	if mapUser["wallet"].([]map[string]interface{})[0]["WALLET_TYPE"].(string) != "K" { return 8217 }

	// 퍼소나 데이타를 생성한다
	ColumnData := make([]_PersonaData, 0)
	
	// 항목에 따라 저장할 값을 가져온다
	if reqBody["age"] != nil && len(reqBody["age"].(string)) > 0 { ColumnData = append(ColumnData, _Persona_Single("age", reqBody)...) }
	if reqBody["sex"] != nil && len(reqBody["sex"].(string)) > 0 { ColumnData = append(ColumnData, _Persona_Single("sex", reqBody)...) }
	if reqBody["job"] != nil && len(reqBody["job"].(map[string]interface{})) > 0 { ColumnData = append(ColumnData, _Persona_Range("job", []string{"L", "M"}, reqBody)...) }
	if reqBody["character"] != nil && len(reqBody["character"].(map[string]interface{})) > 0 { ColumnData = append(ColumnData, _Persona_Range("character", []string{"C", "R", "J", "L"}, reqBody)...) }
	if reqBody["hobby"] != nil && len(reqBody["hobby"].([]interface{})) > 0 { ColumnData = append(ColumnData, _Persona_Array("hobby", reqBody)...) }
	if reqBody["married"] != nil && len(reqBody["married"].(string)) > 0 { ColumnData = append(ColumnData, _Persona_Single("married", reqBody)...) }
	if reqBody["social"] != nil && len(reqBody["social"].([]interface{})) > 0 { ColumnData = append(ColumnData, _Persona_Array("social", reqBody)...) }
	if reqBody["social_usage"] != nil && len(reqBody["social_usage"].(string)) > 0 { ColumnData = append(ColumnData, _Persona_Single("social_usage", reqBody)...) }
	if reqBody["game"] != nil && len(reqBody["game"].([]interface{})) > 0 { ColumnData = append(ColumnData, _Persona_Array("game", reqBody)...) }
	if reqBody["game_genre"] != nil && len(reqBody["game_genre"].([]interface{})) > 0 { ColumnData = append(ColumnData, _Persona_Array("game_genre", reqBody)...) }

	// 입력된 데이타가 없다면
	if len(ColumnData) == 0 { return 8028 }

	// 등록된 내역이 있는지 체크한다
	var count int64
	err = db.QueryRow("SELECT count(USER_KEY) FROM USER_PERSONA WHERE USER_KEY = '" + userkey + "'").Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			count = 0
		} else {
			global.FLog.Println(err)
			return 9901
		}
	}
	
	// 등록 여부에 따라
	if count == 0 {

		// 칼럼부와 데이타부를 생성한다
		var columns string = "USER_KEY,"
		var values string = "'" + userkey + "',"
		for _, data := range ColumnData {
			columns = columns + data.Col + ","
			values = values + "'" + data.Val + "',"
		}
		columns = columns + "UPDATE_TIME"
		values = values + "sysdate"

		// 데이타를 삽입한다
		query := "INSERT INTO USER_PERSONA (" + columns + ") VALUES (" + values + ")"
		_, err = db.Exec(query)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}

		// 에러드랍을 전송한다
		kas, err := common.KAS_Transfer("D:" + userkey,
										adminVar.Wallet.Exchange.Address,
										mapUser["wallet"].([]map[string]interface{})[0]["ADDRESS"].(string),
										fmt.Sprintf("%f", adminVar.Reword.Persona),
										adminVar.Wallet.Exchange.Type,
										adminVar.Wallet.Exchange.CertInfo)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}
		global.FLog.Println(kas)

		resBody["OBSR"] = adminVar.Reword.Persona

	} else {

		// 데이타를 수정할 SQL을 생성한다
		query := "UPDATE USER_PERSONA SET "
		for _, data := range ColumnData {
			query = query + data.Col + "='" + data.Val + "',"
		}
		query = query + "UPDATE_TIME=sysdate "
		query = query + "WHERE USER_KEY = '" + userkey + "'"

		// 데이타를 갱신한다
		_, err = db.Exec(query)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}

		resBody["OBSR"] = 0
	}

	return 0
}

func _Persona_Single(key string, reqBody map[string]interface{}) ([]_PersonaData) {

	arrData := []_PersonaData{ _PersonaData{strings.ToUpper(key), reqBody[key].(string)} }
	return arrData
}

func _Persona_Array(key string, reqBody map[string]interface{}) ([]_PersonaData) {

	var list string

	for _, h := range reqBody[key].([]interface{}) {
		list = list + h.(string) + ","
	}
	if len(list) > 0 { list = list[0:len(list)-1] }

	arrData := []_PersonaData{ _PersonaData{strings.ToUpper(key), list} }
	return arrData
}

func _Persona_Range(key string, sub []string, reqBody map[string]interface{}) ([]_PersonaData) {

	arrData := make([]_PersonaData, 0)

	for _, k := range sub {
		if reqBody[key].(map[string]interface{})[k] != nil && len(reqBody[key].(map[string]interface{})[k].(string)) > 0 {
			arrData = append(arrData, _PersonaData{strings.ToUpper(key) + "_" + k, reqBody[key].(map[string]interface{})[k].(string)})
		}
	}

	return arrData
}
