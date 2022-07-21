package global

import (
	"log"
)

// Config
type ServerConfig struct {

	Service struct {
		Name			string	`json:"name" binding:"required"`
		Mode			string	`json:"mode" binding:"required"`
		APIKey			string	`json:"api_key" binding:"required"`
		APISecret		string	`json:"api_secret" binding:"required"`
		Timezone		string	`json:"timezone" binding:"required"`
		AccountPool		string	`json:"klaytn_account_pool" binding:"required"`
	} `json:"service" binding:"required"`

	WWW struct {
		HttpHost 		string	`json:"http_host" binding:"required"`
		HttpSSLChain 	string	`json:"http_ssl_chain" binding:"required"`
		HttpSSLPrivKey 	string	`json:"http_ssl_privkey" binding:"required"`
	} `json:"www" binding:"required"`

	Database struct {
		Driver       	string `json:"driver" binding:"required"`
		Name         	string `json:"name" binding:"required"`
		MaxOpenConns 	int    `json:"max_open_conns binding:"required"`
		MaxIdleConns 	int    `json:"max_idle_conns binding:"required"`
		WS_Name         string `json:"ws_name" binding:"required"`
		WS_MaxOpenConns int    `json:"ws_max_open_conns binding:"required"`
		WS_MaxIdleConns int    `json:"ws_max_idle_conns binding:"required"`
	} `json:"database" binding:"required"`

	Redis struct {
		Host			string	`json:"host" binding:"required"`
		Password		string	`json:"password" binding:"required"`
	} `json:"redis" binding:"required"`

	Connector struct {
		KASInquiryHost  string	`json:"KAS_inquiry_host" binding:"required"`
		KASTransactHost	string	`json:"KAS_transact_host" binding:"required"`
		SyncRedisHost	string	`json:"sync_redis_host" binding:"required"`
	} `json:"connector" binding:"required"`

	APIs struct {
		WSApi			string	`json:"ws_api" binding:"required"`
		WSKey			string	`json:"ws_key" binding:"required"`
		WSSecret		string	`json:"ws_secret" binding:"required"`
	} `json:"apis" binding:"required"`
}

// Admin Config
type AdminConfig struct {
	Snap struct {
		Interval		int64	`json:"interval" binding:"required"`
		CheckTime		int64	`json:"check_time" binding:"required"`
		CheckRange		float64	`json:"check_range" binding:"required"`
	} `json:"snap" binding:"required"`

	Label struct {
		InquiryTime1	int64	`json:"inqtime1" binding:"required"`
		InquiryTime2	int64	`json:"inqtime2" binding:"required"`
		MaxPerSnap		int64	`json:"max_per_snap" binding:"required"`
		MaxTime			int64	`json:"max_time" binding:"required"`
		AddAdLabel		int64	`json:"add_ad_label" binding:"required"`
	} `json:"label" binding:"required"`

	Reword struct {
		Snap			float64	`json:"snap" binding:"required"`
		Label			float64	`json:"label" binding:"required"`
		LabelEtc		float64	`json:"label_etc" binding:"required"`
		OBSPPerDay		float64	`json:"obsp_per_day" binding:"required"`
		AutoExchange	float64	`json:"auto_exchange" binding:"required"`
	} `json:"reword" binding:"required"`

	Wallet struct {
		Exchange struct {
			Address		string	`json:"address" binding:"required"`
			Type		string	`json:"type" binding:"required"`
			CertInfo	string	`json:"cert" binding:"required"`
		} `json:"exchange" binding:"required"`
		Withdraw struct {
			Address		string	`json:"address" binding:"required"`
			Type		string	`json:"type" binding:"required"`
			CertInfo	string	`json:"cert" binding:"required"`
		} `json:"withdraw" binding:"required"`
	} `json:"wallet" binding:"required"`

	TxFee struct {
		Exchange struct {
			Coin		string	`json:"coin" binding:"required"`
			Fee			float64	`json:"fee" binding:"required"`
		} `json:"exchange" binding:"required"`
		Withdraw struct {
			Coin		string	`json:"coin" binding:"required"`
			Fee			float64	`json:"fee" binding:"required"`
		} `json:"withdraw" binding:"required"`
	} `json:"txfee" binding:"required"`
}

// Header Struct
type HeaderParameter struct {
	XNonce     			int64 	`header:"X-PHOTOAPP-NONCE" binding:"required"`
	XAccess    			string	`header:"X-PHOTOAPP-ACCESS" binding:"required"`
	XSignature 			string	`header:"X-PHOTOAPP-SIGNATURE" binding:"required"`
}

// Tr Header Struct
//type TrHeader struct {
//	TrID 				string	`json:"trid" binding:"required"`
//	Key  				string	`json:"key" binding:"required"`
//	Lang 				string	`json:"lang" binding:"required"`
//}


// Const Valiable
const ConfigFile string = "config/photoApp.json"

// Send Code
const SendCodeExpireSecs int = 120
const SendCodeMaxErrors int = 5
const SendCodeBlockSecs int = 86400

//const LoginMaxErrors int = 5
//const LoginBlockSecs int = 86400

// Global Variable
var Config ServerConfig
var FLog *log.Logger

