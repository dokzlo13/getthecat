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


func ServeStaticImg(ImgDB ImgDB, mode string) func(c *gin.Context) {
	var extraHeaders map[string]string
	switch mode {
	case "image":
		extraHeaders = map[string]string{
				"Content-Disposition": fmt.Sprintf("attachment; filename=\"%s.png\"", ImgDB.Prefix),
			}
	case "attachment":
		extraHeaders = map[string]string{}
	}

	return func(c *gin.Context) {
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
		return
	}
}

func main(){
	rand.Seed(time.Now().Unix())
	configpath := flag.String("conf", "config.yaml", "Config file for server")
	flag.Parse()


	conf, err := LoadConfig(*configpath)
	if err != nil {
		log.Fatalln("Error reading conf")
	}

	setupLogs(conf.Debug, conf.Logfile)

	var Api Searhcer

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

	Watcher = NewImgWatcher(db, conf.WatcherConf.MinimalAviable, conf.WatcherConf.MaximumUses, conf.WatcherConf.Checktime, conf.Debug)

	router := gin.New()
	router.Use(
		gin.LoggerWithWriter(log.StandardLogger().Writer()),
		gin.Recovery(),
	)


	//var ServeFunction func(c *gin.Context)


	ImgDbs = make(map[string]ImgDB, len(conf.Endpoints))

	for _, endpoint := range conf.Endpoints {
		log.Printf("Initalizing serve for \"%s\"", endpoint)
		ImgDbs[endpoint] = NewImgDB(Api, conf.ImgFolder, endpoint)
		go Watcher.WatchImages(ImgDbs[endpoint])
		router.GET("/" + endpoint, ServeStaticImg(ImgDbs[endpoint], conf.ServingConf.ServingType))
	}

	log.Warningln(ImgDbs)
	router.Run("0.0.0.0:8080")

}