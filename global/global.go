package global

import (
	"log"
	"time"
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
		User         	string `json:"user" binding:"required"`
		Password        string `json:"password" binding:"required"`
		ConnectString   string `json:"connectString" binding:"required"`
		MaxOpenConns 	int    `json:"max_open_conns binding:"required"`
		MaxIdleConns 	int    `json:"max_idle_conns binding:"required"`
		WS_Name         string `json:"ws_name" binding:"required"`
		WS_MaxOpenConns int    `json:"ws_max_open_conns binding:"required"`
		WS_MaxIdleConns int    `json:"ws_max_idle_conns binding:"required"`
	} `json:"database" binding:"required"`

	Redis struct {
		Host			string	`json:"host" binding:"required"`
		Password		string	`json:"password" binding:"required"`
		MaxIdleConns 	int    `json:"max_idle_conns binding:"required"`
		MaxActiveConns 	int    `json:"max_active_conns binding:"required"`
	} `json:"redis" binding:"required"`

	Connector struct {
		KASInquiryHost  string	`json:"KAS_inquiry_host" binding:"required"`
		KASTransactHost	string	`json:"KAS_transact_host" binding:"required"`
		SyncRedisHost	string	`json:"sync_redis_host" binding:"required"`
		MQTTPusherHost	string	`json:"MQTT_pusher_host" binding:"required"`
		AddressHost		string	`json:"address_host" binding:"required"`
	} `json:"connector" binding:"required"`

	APIs struct {
		WSApi			string	`json:"ws_api" binding:"required"`
		WSKey			string	`json:"ws_key" binding:"required"`
		WSSecret		string	`json:"ws_secret" binding:"required"`
		MailApi			string	`json:"mail_api" binding:"required"`
		MailKey			string	`json:"mail_key" binding:"required"`
		MailSecret		string	`json:"mail_secret" binding:"required"`
		SMSApi			string	`json:"sms_api" binding:"required"`
		SMSUser			string	`json:"sms_user" binding:"required"`
		SMSKey			string	`json:"sms_key" binding:"required"`
		SMSSender		string	`json:"sms_sender" binding:"required"`
		GabiaSMSUser	string	`json:"gabia_sms_user" binding:"required"`
		GabiaSMSKey		string	`json:"gabia_sms_key" binding:"required"`
		GabiaSMSSender	string	`json:"gabia_sms_sender" binding:"required"`
		NCloudAccessKey	string	`json:"ncloud_access_key" binding:"required"`
		NCloudSecretKey	string	`json:"ncloud_secret_key" binding:"required"`
		NCloudServiceID	string	`json:"ncloud_service_id" binding:"required"`
		NCloudSMSSender	string	`json:"ncloud_sms_sender" binding:"required"`
		OvalAPIKey		string	`json:"oval_apikey" binding:"required"`
		OvalClientId	string	`json:"oval_client_id" binding:"required"`
		OvalSMSSender	string	`json:"oval_sms_sender" binding:"required"`
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
		Persona			float64	`json:"persona" binding:"required"`
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
		Marketing struct {
			Address		string	`json:"address" binding:"required"`
			Type		string	`json:"type" binding:"required"`
			CertInfo	string	`json:"cert" binding:"required"`
		} `json:"marketing" binding:"required"`
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

// OBSR Decimal Point
const OBSR_PDesz int = 8

// Communication Class
const Comm_DataStore string = "DataStore"

// Send Code
const SendCodeExpireSecs int = 120
const SendCodeMaxErrors int = 5
const SendCodeBlockSecs int = 86400

//const LoginMaxErrors int = 5
//const LoginBlockSecs int = 86400

const DBContextTimeout time.Duration = 5

// Global Variable
var Config ServerConfig
var FLog *log.Logger
