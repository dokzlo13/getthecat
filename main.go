package main

import (
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"math/rand"
	"net/http"
	"time"
)

var (
	Watcher ImgWatcher
	ImgDbs  map[string]ImgDB
	db *gorm.DB
	)


func ServeStaticImg(ImgDB ImgDB, conf ServingConf) func(c *gin.Context) {
	var extraHeaders map[string]string
	switch conf.Filetype {
	case "attachment":
		extraHeaders = map[string]string{
				"Content-Disposition": fmt.Sprintf("attachment; filename=\"%s.png\"", ImgDB.Prefix),
			}
	case "image":
		extraHeaders = map[string]string{}
	}

	var responder func(c *gin.Context)

	switch conf.Mode {
	case "cache":
		responder = func(c *gin.Context) {
			imginfo, err := Watcher.GetImg(ImgDB)
			if err != nil {
				log.Errorf("No aviable file found with err \"%v\"", err)
				c.AbortWithError(404, err)
				return
			}

			img, err := Watcher.GetFile(ImgDB, imginfo.ID)
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
			c.DataFromReader(http.StatusOK, imginfo.Filesize, "image/png", img, extraHeaders)
		}
	case "proxy":
		responder = func(c *gin.Context) {
			imginfo, err := Watcher.GetImg(ImgDB)
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
	//case "google":
	//	log.Println("Using GOOGLE")
	//	Api = NewGoogleAPI(conf.Auth.ApiKey, conf.Auth.GoogleCX)
	case "flickr":
		log.Println("Using FLICKR")
		Api = NewFlickrApi(conf.Auth.ApiKey, conf.WatcherConf.MinimalAviable)
	default:
		log.Fatalf("Wrong engine \"%s\" for GetTheCat", conf.Mode)
	}

	db, _ = ConnectDB(conf.DbPath)
	defer db.Close()

	Watcher = NewImgWatcher(db, conf.WatcherConf, conf.Debug)

	router := gin.New()
	router.Use(
		gin.LoggerWithWriter(log.StandardLogger().Writer()),
		gin.Recovery(),
	)

	ImgDbs = make(map[string]ImgDB, len(conf.Endpoints))

	for _, endpoint := range conf.Endpoints {
		log.Printf("Initalizing serve for \"%s\"", endpoint)
		ImgDbs[endpoint] = NewImgDB(Api, conf.ImgFolder, endpoint)
		go Watcher.WatchImages(ImgDbs[endpoint])
		router.GET("/" + endpoint, ServeStaticImg(ImgDbs[endpoint], conf.ServingConf))
	}

	router.Run("0.0.0.0:8080")

}