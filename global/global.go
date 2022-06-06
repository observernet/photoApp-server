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
}

// Admin Config
type AdminConfig struct {
	Snap struct {
		Interval		int64	`json:"interval" binding:"required"`
		CheckTime		int64	`json:"check_time" binding:"required"`
		CheckRange		float64	`json:"check_range" binding:"required"`
	} `json:"snap" binding:"required"`

	Label struct {
		Inquiry			int64	`json:"inquiry" binding:"required"`
		MaxPerSnap		int64	`json:"max_per_snap" binding:"required"`
		MaxTime			int64	`json:"max_time" binding:"required"`
	} `json:"label" binding:"required"`
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

