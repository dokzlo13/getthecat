package main

import (
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var Watcher ImgWatcher

func randrange(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}

func SetupCloseHandler() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\r- Ctrl+C pressed in Terminal, shutting down...")
		Watcher.Sync()
		if err := Watcher.Cache.Flush(); err != nil {
			log.Errorln("Error cleaning cache!")
		} else {
			log.Infoln("Cache cleaned closed!")
		}
		if err := Watcher.DB.Debug().Close(); err != nil {
			log.Errorln("Error closing DB, database may be corrupted")
		} else {
			log.Infoln("Database closed!")
		}
		os.Exit(0)
	}()
}

func main() {
	var Api Searhcer

	rand.Seed(time.Now().Unix())
	configpath := flag.String("conf", "config.yaml", "Config file for server")
	flag.Parse()

	conf, err := LoadConfig(*configpath)
	if err != nil {
		log.Fatalf("Error Loading config-file: \"%s\"", err)
	}

	setupLogs(conf.Debug, conf.Logfile)

	switch conf.Mode {
	case "google":
		log.Warningln("Using GOOGLE")
		Api = NewGoogleAPI(conf.Auth.ApiKey, conf.Auth.GoogleCX)
	case "flickr":
		log.Warningln("Using FLICKR")
		Api = NewFlickrApi(conf.Auth.ApiKey, conf.WatcherConf.MinimalAviable)
	default:
		log.Fatalf("Wrong engine \"%s\" for GetTheCat", conf.Mode)
	}

	db, err := ConnectDB(conf.DbPath)
	if err != nil {
		log.Fatalf("Cannot connect to DB %s with error \"%s\"", conf.DbPath, err)
	}
	//defer db.Close()

	Watcher = NewImgWatcher(db, conf.WatcherConf, conf.Debug)
	SetupCloseHandler()

	Watcher.RemoveEmptyFiles()

	router := gin.New()
	router.Use(
		gin.LoggerWithWriter(log.StandardLogger().Writer()),
		gin.Recovery(),
	)

	var api *gin.RouterGroup
	if apipath := conf.ServingConf.ApiPath; apipath != "" {
		api = router.Group(apipath)
	} else {
		api = router.Group("")
	}

	for _, endpoint := range conf.Endpoints {
		log.Printf("Initalizing serve for \"%s\"", endpoint)
		//ImgDbs[endpoint] =
		Imdb := NewImgDB(Api, conf.ImgFolder, endpoint)
		go Watcher.WatchImages(Imdb)
		subgroup := api.Group(endpoint)
		subgroup.GET("/info/new", ServeActualImgInfo(endpoint))
		subgroup.GET("/info/rand", ServeRandomImgInfo(endpoint))
		subgroup.GET("/info/static/:id", ServeImgInfo(endpoint))

		subgroup.GET("/img/new", ServeActualImg(endpoint, conf.ServingConf))
		subgroup.GET("/img/rand", ServeRandomImg(endpoint, conf.ServingConf))
		subgroup.GET("/img/static/:id", ServeImg(endpoint, conf.ServingConf))
	}

	go Watcher.StartSync()
	router.Run("0.0.0.0:8080")

}
