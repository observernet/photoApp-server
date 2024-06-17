package RestV10

import (
	"fmt"
	"time"
	"context"
	"strconv"
	//"encoding/json"
	
	"photoApp-server/global"
	"photoApp-server/common"

	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
	"github.com/godror/godror"
)

// ReqData - loginkey: 로그인키
//         - exclude: 제외스냅키리스트
// ResData - list: 스냅리스트 {snapkey: 스냅키, name: 스냅제공자, photo: 스냅제공자프로필, lat:위도, lng: 경도, url: 이미지링크, time: 스냅시간, labels: 현재라벨수}
func TR_SnapList(c *gin.Context, db *sql.DB, rds redis.Conn, lang string, reqData map[string]interface{}, resBody map[string]interface{}) int {

	ctx, cancel := context.WithTimeout(c, global.DBContextTimeout * time.Second)
	defer cancel()

	userkey := reqData["key"].(string)
	reqBody := reqData["body"].(map[string]interface{})

	// check input
	if reqBody["loginkey"] == nil { return 9003 }
	curtime_Sec := time.Now().Unix()

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
	var adminVar global.AdminConfig
	if adminVar, err = common.GetAdminVar(rds); err != nil {
		global.FLog.Println(err)
		return 9901
	}


	// 계정 상태 및 로그인 정보를 체크한다
	if mapUser["info"].(map[string]interface{})["STATUS"].(string) != "V" { return 8013 }
	if mapUser["login"].(map[string]interface{})["loginkey"].(string) != reqBody["loginkey"].(string) { return 8014 }

	var stmt *sql.Stmt
	var rows *sql.Rows
	var mapRows []map[string]interface{}
	var query, exclude, snap_date, snap_idx string
	var inqTime int64

	// 제외키를 세팅한다
	if reqBody["exclude"] != nil && len(reqBody["exclude"].([]interface{})) > 0 {
		exclude = "  and (A.SNAP_DATE, A.SNAP_IDX) NOT IN ("
		for _, item := range reqBody["exclude"].([]interface{}) {
			snap_date, snap_idx, err = common.GetSnapKey(item.(string))
			if err != nil {
				global.FLog.Println(err)
				return 9003
			}
			exclude = exclude + "(" + snap_date + ", " + snap_idx + "),"
		}
		exclude = exclude[0:len(exclude)-1] + ") "
	}

	// 스냅리스트를 가져올 SQL을 생성한다
	query = "SELECT AB.SNAP_DATE, AB.SNAP_IDX, AB.SNAP_TIME, AB.LATD, AB.LNGD, AB.IMAGE_URL, AB.IMAGE_TYPE, AB.IMAGE_SUB, AB.USER_KEY, AB.NAME, AB.PHOTO, AB.NOTE, AB.ADDR, AB.LABELS, AB.LIKES, AB.MYLIKE " +
			"FROM " +
			"( " +
			"	SELECT A.SNAP_DATE, A.SNAP_IDX, DATE_TO_UNIXTIME(A.SNAP_TIME) SNAP_TIME, A.LATD, A.LNGD, A.IMAGE_URL, A.IMAGE_TYPE, A.IMAGE_SUB, A.USER_KEY, B.NAME, B.PHOTO, " +
			"		   NVL(A.NOTE, ' ') NOTE, NVL(A.ADDR, '::::::') ADDR, " +
			"		   (SELECT count(LABEL_IDX) FROM SNAP_LABEL WHERE SNAP_DATE = A.SNAP_DATE AND SNAP_IDX = A.SNAP_IDX) LABELS, " +
			"          (SELECT count(LABEL_IDX) FROM SNAP_LABEL WHERE SNAP_DATE = A.SNAP_DATE AND SNAP_IDX = A.SNAP_IDX AND USER_KEY = '" + userkey + "') MYLABELS, " +
			"		   (SELECT count(REACTION_IDX) FROM SNAP_REACTION WHERE SNAP_DATE = A.SNAP_DATE AND SNAP_IDX = A.SNAP_IDX AND REACTION_TYPE = 'L') LIKES, " +
			"		   (SELECT count(REACTION_IDX) FROM SNAP_REACTION WHERE SNAP_DATE = A.SNAP_DATE AND SNAP_IDX = A.SNAP_IDX AND REACTION_TYPE = 'L' AND USER_KEY = '" + userkey + "') MYLIKE " +
			"  FROM SNAP A, USER_INFO B " +
			"  WHERE A.USER_KEY = B.USER_KEY " +
		  	"    and A.UPLOAD_STATUS = 'V' " +
		  	"    and A.USER_KEY != '" + userkey + "' " +
			//"    and A.USER_KEY in ('Cne8QAxPR6mp9iqI', 'fUfrqy5ARn3ftQxM') " +
			"    and A.USER_KEY not in (SELECT BLOCK_USER_KEY FROM USER_BLOCK WHERE USER_KEY = '" + userkey + "') " +
		  	"    and A.IS_SHOW = 'Y' " +
		  	exclude +
		    ") AB " +
			"WHERE AB.SNAP_TIME > :1 " +
	  		"  and AB.LABELS < " + strconv.FormatInt(adminVar.Label.MaxPerSnap, 10) + " " +
	  		"  and MYLABELS = 0 " +
		    "ORDER BY DBMS_RANDOM.VALUE "
	if stmt, err = db.PrepareContext(ctx, query); err != nil {
		global.FLog.Println(err)
		return 9901
	}
	defer stmt.Close()

	// 스냅리스트를 가져온다 (1st: 1시간이내, 2nd: 1일이내, 3rd:전체 - 10건 조회될때까지)
	for i := 0; i < 3; i++ {
		if i == 0 {
			inqTime = curtime_Sec - adminVar.Label.InquiryTime1
		} else if i == 1 {
			inqTime = curtime_Sec - adminVar.Label.InquiryTime2
		} else {
			inqTime = 0
		}

		if rows, err = stmt.Query(inqTime); err != nil {
			global.FLog.Println(err)
			return 9901
		}
		defer rows.Close()

		if mapRows, err = common.GetRowsResult(rows, 10); err != nil {
			global.FLog.Println(err)
			return 9901
		}

		if len(mapRows) >= 10 { break }
	}

	list := make([]map[string]interface{}, 0)
	for _, lst := range mapRows {
		snapkey := fmt.Sprintf("%08d%06d", lst["SNAP_DATE"], lst["SNAP_IDX"])
		imageUrl := lst["IMAGE_URL"].(string) + "/Snap/View/L/" + snapkey + "/" + lst["IMAGE_SUB"].(string) + "/" + lst["IMAGE_TYPE"].(string)

		list = append(list, map[string]interface{} {
								"snapkey": snapkey,
								"userkey": lst["USER_KEY"].(string),
								"name":    lst["NAME"].(string),
								"photo":   "https://photoapp.obsr-app.org/Image/View/profile/" + lst["USER_KEY"].(string),
								"lat":     common.GetFloat64FromNumber(lst["LATD"].(godror.Number)),
								"lng":     common.GetFloat64FromNumber(lst["LNGD"].(godror.Number)),
								"url":     imageUrl,
								"time":    common.GetInt64FromNumber(lst["SNAP_TIME"].(godror.Number)) * 1000,
								"labels":  common.GetInt64FromNumber(lst["LABELS"].(godror.Number)),
								"reactions": map[string]interface{} {
									"likes":   common.GetInt64FromNumber(lst["LIKES"].(godror.Number)),
							    	"mylike":  common.GetInt64FromNumber(lst["MYLIKE"].(godror.Number)) },
								"note":    lst["NOTE"].(string),
								"addr":    lst["ADDR"].(string) })
	}
	resBody["list"] = list

	return 0
}
