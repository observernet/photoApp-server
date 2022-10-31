package RestV10

import (
	"bytes"
	"database/sql"
	"time"
	"strconv"
	"strings"
	"net/http"

    "encoding/base64"
	"encoding/json"

	"photoApp-server/global"
	"photoApp-server/common"

	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

// Receive Request & Send Response
func ProcRestV10(c *gin.Context, db *sql.DB, rdp *redis.Pool) {

	var err error
	var header global.HeaderParameter

	// Read Header & Body
	if err = c.ShouldBindHeader(&header); err != nil {
		global.FLog.Println("Header JSON Parse Error")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": 9001, "lang": "E", "message": common.GetErrorMessage("E", 9001)})
		return
	}
	buf := new(bytes.Buffer); buf.ReadFrom(c.Request.Body)
	hbody := buf.String()

	// Check Valid Header & Data
	if ret := CheckHeader(header, hbody); ret > 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": ret, "lang": "E", "message": common.GetErrorMessage("E", ret)})
		return
	}

	// write log
	tr, _ := base64.StdEncoding.DecodeString(hbody)
	global.FLog.Println("Request:", string(tr))

	// request body json to Map
	reqData := make(map[string]interface{})	
	if err = json.Unmarshal(tr, &reqData); err != nil {
		global.FLog.Println("Body JSON Parse Error")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": 9003, "lang": "E", "message": common.GetErrorMessage("E", 9003)})
		return
	}

	// check tr struct
	if reqData["trid"] == nil || reqData["key"] == nil || reqData["lang"] == nil || reqData["body"] == nil {
		global.FLog.Println("Body Struct Error")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": 9003, "lang": "E", "message": common.GetErrorMessage("E", 9003)})
		return
	}

	var res_code int
	var lang string = reqData["lang"].(string)

	// Change Lang
	if strings.EqualFold(lang, "kr") { lang = "K" }
	if strings.EqualFold(lang, "en") { lang = "E" }

	// Redis Connect
	rds := rdp.Get();
	defer rds.Close();

	// switch request
	resBody := make(map[string]interface{})
	switch reqData["trid"] {
		case "login_info": res_code = TR_LoginInfo(c, db, rds, lang, reqData, resBody)
		case "banner": res_code = TR_Banner(c, db, rds, lang, reqData, resBody)
		case "popup": res_code = TR_Popup(c, db, rds, lang, reqData, resBody)
		case "snap_list": res_code = TR_SnapList(c, db, rds, lang, reqData, resBody)
		case "label": res_code = TR_Label(c, db, rds, lang, reqData, resBody)
		case "ad_reword": res_code = TR_AdReword(c, db, rds, lang, reqData, resBody)
		case "fcm_update": res_code = TR_FcmUpdate(c, db, rds, lang, reqData, resBody)
		case "snap": res_code = TR_Snap(c, db, rds, lang, reqData, resBody)
		case "version": res_code = TR_Version(c, db, rds, lang, reqData, resBody)
		case "login": res_code = TR_Login(c, db, rds, lang, reqData, resBody)
		case "logout": res_code = TR_Logout(c, db, rds, lang, reqData, resBody)
		case "wslist": res_code = TR_WSList(c, db, rds, lang, reqData, resBody)
		case "obsp_list": res_code = TR_OBSPList(c, db, rds, lang, reqData, resBody)
		case "obsp_detail": res_code = TR_OBSPDetail(c, db, rds, lang, reqData, resBody)
		case "wallet_obsr": res_code = TR_WalletOBSR(c, db, rds, lang, reqData, resBody)
		case "inout_list": res_code = TR_InoutList(c, db, rds, lang, reqData, resBody)
		case "exchange_info": res_code = TR_ExchangeInfo(c, db, rds, lang, reqData, resBody)
		case "exchange": res_code = TR_Exchange(c, db, rds, lang, reqData, resBody)
		case "withdraw_info": res_code = TR_WithdrawInfo(c, db, rds, lang, reqData, resBody)
		case "withdraw": res_code = TR_Withdraw(c, db, rds, lang, reqData, resBody)
		case "update_wallet": res_code = TR_UpdateWallet(c, db, rds, lang, reqData, resBody)
		case "mysnap_list": res_code = TR_MySnapList(c, db, rds, lang, reqData, resBody)
		case "notice": res_code = TR_Notice(c, db, rds, lang, reqData, resBody)
		case "join": res_code = TR_Join(c, db, rds, lang, reqData, resBody)
		case "check_name": res_code = TR_CheckName(c, db, rds, lang, reqData, resBody)
		case "regist_email": res_code = TR_RegistEmail(c, db, rds, lang, reqData, resBody)
		case "search_user": res_code = TR_SearchUser(c, db, rds, lang, reqData, resBody)
		case "search_passwd": res_code = TR_SearchPasswd(c, db, rds, lang, reqData, resBody)
		case "update_user": res_code = TR_UpdateUser(c, db, rds, lang, reqData, resBody)
		case "update_name": res_code = TR_UpdateName(c, db, rds, lang, reqData, resBody)
		case "persona": res_code = TR_Persona(c, db, rds, lang, reqData, resBody)
		case "joinout": res_code = TR_JoinOut(c, db, rds, lang, reqData, resBody)
		case "userblock": res_code = TR_UserBlock(c, db, rds, lang, reqData, resBody)
		case "userblock_clear": res_code = TR_UserBlockClear(c, db, rds, lang, reqData, resBody)
		case "userblock_list": res_code = TR_UserBlockList(c, db, rds, lang, reqData, resBody)
		default:
			global.FLog.Println("정의되지 않은 TR:", reqData["trid"])
			res_code = 9004
	}

	// Error Occurred
	//if ( res_code > 0 ) {
	//	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": res_code, "lang": lang, "message": common.GetErrorMessage(lang, res_code)})
	//	return
	//}

	// Fill Response TR
	responseTR := make(map[string]interface{})
	responseTR["code"] = res_code
	if res_code > 0 { responseTR["message"] = common.GetErrorMessage(lang, res_code) }
	responseTR["lang"] = lang
	responseTR["trid"] = reqData["trid"]
	if len(resBody) > 0 { responseTR["body"] = resBody }

	// Print Response
	meJson, _ := json.Marshal(responseTR)
	strJson := string(meJson)
	c.String(http.StatusOK, strJson)

	// write log
	global.FLog.Println("Response:", strJson)
}

func CheckHeader(header global.HeaderParameter, reqdata string) int {

	// Check Access Key
	if header.XAccess != global.Config.Service.APIKey {
		return 9001
	}

	// Check Timeout
	timestamp := time.Now().Unix() * 1000;
	if timestamp - header.XNonce > 5000 {
		global.FLog.Println("Header Time Error", header)
		return 9002
	}

	// Check Signatue
	endata := strconv.FormatInt(header.XNonce, 10) + header.XAccess + reqdata
	sign := common.EncyptData(endata, global.Config.Service.APISecret)
	if sign != header.XSignature {
		global.FLog.Println("Header Sign Error", header)
		return 9001
	}

	return 0
}
