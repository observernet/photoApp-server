package RestV10

import (
	"fmt"
	"time"
	"context"
	
	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

type _MySnapList_Labels struct {
	Status_0			int64
	Status_1			int64
	Sky_N				int64
	Sky_Y				int64
	Sky_F				int64
	Rain_N				int64
	Rain_Y				int64
	Rain_S				int64
	Rain_I				int64
	Rain_W				int64
	Wcondi_N			int64
	Wcondi_F			int64
	Wcondi_H			int64
	Wcondi_R			int64
	Wcondi_T			int64
	Wcondi_I			int64
	Calamity_N			int64
	Calamity_R			int64
	Calamity_S			int64
	Calamity_F			int64
	Calamity_L			int64
	Calamity_O			int64
	Calamity_E			int64
	Calamity_W			int64
	Calamity_B			int64
	Calamity_A			int64
	Calamity_H			int64
}

type _MySnapList_Reactions struct {
	Likes				int64
}

// ReqData - 
// ResData - 
func TR_MySnapList(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	ctx, cancel := context.WithTimeout(c, global.DBContextTimeout * time.Second)
	defer cancel()

	userkey := reqData["key"].(string)
	reqBody := reqData["body"].(map[string]interface{})

	// check input
	if reqBody["loginkey"] == nil { return 9003 }

	var err error

	// 유저 정보를 가져온다
	var mapUser map[string]interface{}
	if mapUser, err = common.User_GetInfo(rds, userkey, "info", "login"); err != nil {
		if err == redis.ErrNil {
			return 8015
		} else {
			global.FLog.Println(err)
			return 9901
		}
	}

	// 관리자 정의변수를 가져온다
	//var adminVar global.AdminConfig
	//if adminVar, err = common.GetAdminVar(rds); err != nil {
	//	global.FLog.Println(err)
	//	return 9901
	//}

	// 계정 상태 및 로그인 정보를 체크한다
	if mapUser["info"].(map[string]interface{})["STATUS"].(string) != "V" { return 8013 }
	if mapUser["login"].(map[string]interface{})["loginkey"].(string) != reqBody["loginkey"].(string) { return 8014 }

	// 내 스냅리스트를 가져온다
	query := "SELECT SNAP_DATE, SNAP_IDX, DATE_TO_UNIXTIME(SNAP_TIME), LATD, LNGD, IMAGE_URL, IMAGE_TYPE, IMAGE_SUB, NVL(NOTE, ' ') " +
			 "FROM SNAP " +
			 "WHERE USER_KEY = '" + userkey + "' " +
			 "  and UPLOAD_STATUS = 'V' "
	if reqBody["next"] != nil && len(reqBody["next"].(string)) > 1 { query = query + "  and SNAP_DATE * 1000000 + SNAP_IDX < '" + reqBody["next"].(string) + "'" }
	query = query + "ORDER BY SNAP_TIME desc"
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		global.FLog.Println(err)
		return 9901
	}
	defer rows.Close()

	var snap_date, snap_idx, snap_time int64
	var lat, lng float64
	var image_url, image_type, image_sub, note string
	var count int64

	list := make([]map[string]interface{}, 0)
	for rows.Next() {	
		err = rows.Scan(&snap_date, &snap_idx, &snap_time, &lat, &lng, &image_url, &image_type, &image_sub, &note)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}
		snapkey := fmt.Sprintf("%08d%06d", snap_date, snap_idx)
		imageUrl := image_url + "/Snap/View/S/" + snapkey + "/" + image_sub + "/" + image_type

		// Label 정보를 가져온다
		labels, err := _MySnapList_GetLabels(ctx, db, snap_date, snap_idx)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}

		// 리액션 정보를 가져온다
		reactions, err := _MySnapList_GetReactions(ctx, db, snap_date, snap_idx)
		if err != nil {
			global.FLog.Println(err)
			return 9901
		}

		list = append(list, map[string]interface{} {"snapkey": snapkey, "lat": lat, "lng": lng, "url": imageUrl, "time": snap_time * 1000,
							"labels": map[string]interface{} {
								"state": map[string]interface{} {
									"st0": labels.Status_0,
									"st1": labels.Status_1},
								"cloud": map[string]interface{} {
									"skN": labels.Sky_N,
									"skY": labels.Sky_Y,
									"skF": labels.Sky_F},
								"rain": map[string]interface{} {
									"raN": labels.Rain_N,
									"raY": labels.Rain_Y,
									"raS": labels.Rain_S,
									"raI": labels.Rain_I,
									"raW": labels.Rain_W},
								"nature": map[string]interface{} {
									"wcN": labels.Wcondi_N,
									"wcF": labels.Wcondi_F,
									"wcH": labels.Wcondi_H,
									"wcR": labels.Wcondi_R,
									"wcT": labels.Wcondi_T,
									"wcI": labels.Wcondi_I},
								"calamity": map[string]interface{} {
									"caN": labels.Calamity_N,
									"caR": labels.Calamity_R,
									"caS": labels.Calamity_S,
									"caF": labels.Calamity_F,
									"caL": labels.Calamity_L,
									"caO": labels.Calamity_O,
									"caE": labels.Calamity_E,
									"caW": labels.Calamity_W,
									"caB": labels.Calamity_B,
									"caA": labels.Calamity_A,
									"caH": labels.Calamity_H}},
							"reactions": map[string]interface{} {
								"likes": reactions.Likes},
							"note": note })

		count++
		if count >= 30 {
			resBody["next"] = snapkey
			break
		}
	}
	resBody["list"] = list

	return 0
}

func _MySnapList_GetLabels(ctx context.Context, db *sql.DB, snap_date int64, snap_idx int64) (_MySnapList_Labels, error) {

	var err error
	var labels _MySnapList_Labels

	query := "SELECT STATUS, SKY, RAIN, WCONDI, CALAMITY, IS_ETC " +
			 "FROM SNAP_LABEL " +
			 "WHERE SNAP_DATE = " + fmt.Sprintf("%d", snap_date) +
			 "  and SNAP_IDX = " + fmt.Sprintf("%d", snap_idx)
	rows, err := db.QueryContext(ctx, query)
	if err != nil { return labels, err }
	defer rows.Close()

	var status, sky, rain, wcondi, calamity, is_etc string
	for rows.Next() {	
		err = rows.Scan(&status, &sky, &rain, &wcondi, &calamity, &is_etc)
		if err != nil { return labels, err }

		if status == "0" { labels.Status_0++ }
		if status == "1" { labels.Status_1++ }
		if sky == "N" { labels.Sky_N++ }
		if sky == "Y" { labels.Sky_Y++ }
		if sky == "F" { labels.Sky_F++ }
		if rain == "N" { labels.Rain_N++ }
		if rain == "Y" { labels.Rain_Y++ }
		if rain == "S" { labels.Rain_S++ }
		if rain == "I" { labels.Rain_I++ }
		if rain == "W" { labels.Rain_W++ }

		if is_etc == "Y" {
			if wcondi == "N" { labels.Wcondi_N++ }
			if wcondi == "F" { labels.Wcondi_F++ }
			if wcondi == "H" { labels.Wcondi_H++ }
			if wcondi == "R" { labels.Wcondi_R++ }
			if wcondi == "T" { labels.Wcondi_T++ }
			if wcondi == "I" { labels.Wcondi_I++ }
			if calamity == "N" { labels.Calamity_N++ }
			if calamity == "R" { labels.Calamity_R++ }
			if calamity == "S" { labels.Calamity_S++ }
			if calamity == "F" { labels.Calamity_F++ }
			if calamity == "L" { labels.Calamity_L++ }
			if calamity == "O" { labels.Calamity_O++ }
			if calamity == "E" { labels.Calamity_E++ }
			if calamity == "W" { labels.Calamity_W++ }
			if calamity == "B" { labels.Calamity_B++ }
			if calamity == "A" { labels.Calamity_A++ }
			if calamity == "H" { labels.Calamity_H++ }
		}
	}

	return labels, nil
}

func _MySnapList_GetReactions(ctx context.Context, db *sql.DB, snap_date int64, snap_idx int64) (_MySnapList_Reactions, error) {

	var err error
	var reactions _MySnapList_Reactions

	query := "SELECT REACTION_TYPE " +
			 "FROM SNAP_REACTION " +
			 "WHERE SNAP_DATE = " + fmt.Sprintf("%d", snap_date) +
			 "  and SNAP_IDX = " + fmt.Sprintf("%d", snap_idx)
	rows, err := db.QueryContext(ctx, query)
	if err != nil { return reactions, err }
	defer rows.Close()

	var rtype string
	for rows.Next() {	
		err = rows.Scan(&rtype)
		if err != nil { return reactions, err }

		if rtype == "L" { reactions.Likes++ }
	}

	return reactions, nil
}
