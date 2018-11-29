package main

import (
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	Watcher ImgWatcher
	//ImgDbs  map[string]ImgDB
	db *gorm.DB
	)

func setHeaders(filetype string, prefix string) map[string]string {
	var extraHeaders map[string]string
	switch filetype {
	case "attachment":
		extraHeaders = map[string]string{
			"Content-Disposition": fmt.Sprintf("attachment; filename=\"%s.png\"", prefix),
		}
	case "image":
		extraHeaders = map[string]string{}
	}
	return extraHeaders

}


func ServeImgInfo(prefix string) func(c *gin.Context) {
	return func(c *gin.Context) {
		imginfo, err := Watcher.GetImgById(prefix, c.Param("id"), false)
		if err != nil {
			log.Errorf("No aviable file found with err \"%v\"", err)
			c.AbortWithError(404, err)
			return
		}
		c.JSON(200, imginfo)
	}
}

func ServeRandomImgInfo(prefix string) func(c *gin.Context) {
	return func(c *gin.Context) {
		imginfo, err := Watcher.GetImg(prefix, false)
		if err != nil {
			log.Errorf("No aviable file found with err \"%v\"", err)
			c.AbortWithError(404, err)
			return
		}
		c.JSON(200, imginfo)
	}
}

func ServeImg(prefix string, conf ServingConf) func(c *gin.Context) {
	headers := setHeaders(conf.Filetype, prefix)
	var responder func(c *gin.Context)
	switch conf.Mode {
	case "cache":
		responder = func(c *gin.Context) {
			imginfo, err := Watcher.GetImgById(prefix, c.Param("id"), true)
			if err != nil {
				log.Errorf("No requested file found with err \"%v\"", err)
				c.AbortWithError(404, err)
				return
			}

			img, err := Watcher.GetFile(prefix, imginfo.ID)
			if err != nil {
				log.Errorf("Error opening file %v, %v", img, err)
				c.AbortWithError(502, err)
				return
			}

			if img == nil {
				log.Errorf("Empty file %v, %v", img, err)
				c.AbortWithError(502, err)
				return
			}
			c.DataFromReader(http.StatusOK, imginfo.Filesize, "image/png", img, headers)
		}
	case "proxy":
		responder = func(c *gin.Context) {
			imginfo, err := Watcher.GetImgById(prefix, c.Param("id"), true)
			if err != nil {
				log.Errorf("No aviable file found with err \"%v\"", err)
				c.AbortWithError(404, err)
				return
			}
			c.Redirect(303, imginfo.Origin)
		}
	}

	return responder
}

func ServeRandomImg(prefix string, conf ServingConf) func(c *gin.Context) {
	headers := setHeaders(conf.Filetype, prefix)
	var responder func(c *gin.Context)
	switch conf.Mode {
	case "cache":
		responder = func(c *gin.Context) {
			imginfo, err := Watcher.GetImg(prefix, true)
			if err != nil {
				log.Errorf("No aviable file found with err \"%v\"", err)
				c.AbortWithError(404, err)
				return
			}

			img, err := Watcher.GetFile(prefix, imginfo.ID)
			if err != nil {
				log.Errorf("Error opening file %v, %v", img, err)
				c.AbortWithError(502, err)
				return
			}

			if img == nil {
				log.Errorf("Empty file %v, %v", img, err)
				c.AbortWithError(502, err)
				return
			}
			c.DataFromReader(http.StatusOK, imginfo.Filesize, "image/png", img, headers)
		}
	case "proxy":
		responder = func(c *gin.Context) {
			imginfo, err := Watcher.GetImg(prefix, true)
			if err != nil {
				log.Errorf("No aviable file found with err \"%v\"", err)
				c.AbortWithError(404, err)
				return
			}
			c.Redirect(303, imginfo.Origin)
		}
	}

	return responder
}


func SetupCloseHandler() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\r- Ctrl+C pressed in Terminal, shutting down...")
		Watcher.Sync()
		err := db.Debug().Close()
		if err != nil {
			log.Errorln("Error closing DB, database may be corrupted")
		} else {
			log.Infoln("Database closed!")
		}
		os.Exit(0)
	}()
}


func main(){
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

	db, err = ConnectDB(conf.DbPath)
	if err != nil {
		log.Fatalf("Cannot connect to DB %s with error \"%s\"", conf.DbPath, err)
	}
	//defer db.Close()
	SetupCloseHandler()

	Watcher = NewImgWatcher(db, conf.WatcherConf, conf.Debug)

	router := gin.New()
	router.Use(
		gin.LoggerWithWriter(log.StandardLogger().Writer()),
		gin.Recovery(),
	)

	var api *gin.RouterGroup
	//ImgDbs = make(map[string]ImgDB, len(conf.Endpoints))
	if apipath:=conf.ServingConf.ApiPath; apipath != "" {
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
		subgroup.GET("/info", ServeRandomImgInfo(endpoint))
		subgroup.GET("/info/:id", ServeImgInfo(endpoint))
		subgroup.GET("/img", ServeRandomImg(endpoint, conf.ServingConf))
		subgroup.GET("/img/:id", ServeImg(endpoint, conf.ServingConf))
	}

	go Watcher.StartSync()
	router.Run("0.0.0.0:8080")

}