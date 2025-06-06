package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"log"
	"time"
	"runtime"
	"io"
	"syscall"

	"photoApp-server/global"
	//"photoApp-server/common"
	"photoApp-server/RestV10"

	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/godror/godror"
	"github.com/godror/godror/dsn"
	"github.com/gomodule/redigo/redis"
)

func main() {

	time.Now()		// 이해할수 없지만, 호출 한번해야 타임존이 제대로 설정된다 ㅡㅡ
	//os.Setenv("TZ", global.Config.Service.Timezone)
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Load Config File
	LoadServerConfig(global.ConfigFile)
	
	// Logger Setting
	fLog := SettingLogger(); defer fLog.Close()
	
	// Connect Database
	db := ConnectDatabase(); defer db.Close()
	rdp := ConnectRedis(); defer rdp.Close()

	// set gin framework
	var router *gin.Engine
	
	if global.Config.Service.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
		gin.DisableConsoleColor()
		
		// Gin Log File Setting
		fgin, err := os.OpenFile("log/" + global.Config.Service.Name + "-gin." + time.Now().Format("20060102") + ".log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}
		defer fgin.Close()
		gin.DefaultWriter = io.MultiWriter(fgin)

		router = gin.New()
		router.Use(gin.LoggerWithFormatter(ginLogFormatter()))
		router.Use(gin.Recovery())
		router.Use(CORSMiddleware())
	} else {
		router = gin.Default()
	}
	
	// Gin Routing
	router.POST("/V10", func(c *gin.Context) {
		RestV10.ProcRestV10(c, db, rdp)
	})
	
	s := &http.Server{
		Addr:           global.Config.WWW.HttpHost,
		Handler:        router,
		ReadTimeout:    3 * time.Second,
		WriteTimeout:   3 * time.Second,
		IdleTimeout:	5 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	s.ListenAndServeTLS(global.Config.WWW.HttpSSLChain, global.Config.WWW.HttpSSLPrivKey)
	//s.ListenAndServe()

	//router.RunTLS(global.Config.WWW.HttpHost, global.Config.WWW.HttpSSLChain, global.Config.WWW.HttpSSLPrivKey)
}

func ginLogFormatter() func(param gin.LogFormatterParams) string {
    return func(param gin.LogFormatterParams) string {
        return fmt.Sprintf("[%s] %s | %s | %s | %s | %d | %s | %s | %s\n",
            param.TimeStamp.Format("2006-01-02 15:04:05"),
			param.ClientIP,
            param.Method,
            param.Path,
            param.Request.Proto,
            param.StatusCode,
            param.Latency,
            param.Request.UserAgent(),
            param.ErrorMessage,
        )
    }
}

func CORSMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, Origin")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "POST")
		c.Header("Content-Type", "application/json")

        //if c.Request.Method == "OPTIONS" {
        //    c.AbortWithStatus(204)
        //    return
        //}

        c.Next()
    }
}

func LoadServerConfig(filename string) error {

	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&global.Config)
	if err != nil {
		panic(err)
	}

	return nil
}

func SettingLogger() *os.File {

	var file *os.File
	var err error

	if global.Config.Service.Mode == "release" {
		os.Mkdir("log", 0755)
		file, err = os.OpenFile("log/" + global.Config.Service.Name + "." + time.Now().Format("20060102") + ".log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}
		syscall.Dup2(int(file.Fd()), 2)
		global.FLog = log.New(file, "", log.Ldate|log.Ltime|log.Lshortfile)
	} else {
		global.FLog = log.New(os.Stdout, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile)
	}

	return file
}

func ConnectDatabase() (*sql.DB) {

	var db *sql.DB
	//var err error

	dbConf := godror.ConnectionParams{
		CommonParams: dsn.CommonParams{
			Username: global.Config.Database.User,
			Password: dsn.NewPassword(global.Config.Database.Password),
			Timezone: time.Local,
		},
		ConnParams: godror.ConnParams{
		 	ConnClass: "PHOTOAPP",
		},
		PoolParams: godror.PoolParams{
			MinSessions:      global.Config.Database.MaxIdleConns,
			MaxSessions:      global.Config.Database.MaxOpenConns,
			SessionIncrement: 1,
			SessionTimeout:   1 * time.Minute,
			WaitTimeout:	  5 * time.Second,
			MaxLifeTime:      5 * time.Minute,
		},
	}
	dbConf.ConnectString = global.Config.Database.ConnectString
	db = sql.OpenDB(godror.NewConnector(dbConf))
	db.SetMaxOpenConns(global.Config.Database.MaxOpenConns)
	db.SetMaxIdleConns(0)
	db.SetConnMaxLifetime(0)

	return db
}

func ConnectRedis() (*redis.Pool) {

	pool := &redis.Pool {
        MaxIdle: global.Config.Redis.MaxIdleConns,
        MaxActive: global.Config.Redis.MaxActiveConns,
		IdleTimeout: 600 * time.Second,
        Dial: func() (redis.Conn, error) {
            rc, err := redis.Dial("tcp", global.Config.Redis.Host)
            if err != nil { return nil, err }

			_, err = rc.Do("AUTH", global.Config.Redis.Password)
			if err != nil { return nil, err }
            return rc, nil
        },
    }
	return pool
}
