package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"math/rand"
	"net/http"
	"os"
	"time"
)

var (
	Watcher ImgWatcher
	ImgDbs  map[string]ImgDB
	db *gorm.DB
	)


func ServeImg(ImgDB ImgDB) func(c *gin.Context) {
	return func(c *gin.Context) {
		img := Watcher.GetImg(ImgDB)
		f, _ := os.Open(img)
		fi, err := f.Stat()
		if err != nil {
			// Could not obtain stat, handle error
		}
		//extraHeaders := map[string]string{
		//	"Content-Disposition": fmt.Sprintf("attachment; filename=\"%s.png\"", ImgDB.Prefix),
		//}
		//c.DataFromReader(http.StatusOK, fi.Size(), "image/png", f, extraHeaders)
		c.DataFromReader(http.StatusOK, fi.Size(), "image/png", f, map[string]string{})

	}
}

func main(){
	rand.Seed(time.Now().Unix())

	configpath := "config2.yaml"


	conf, err := LoadConfig(configpath)
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
	}

	//DogImg = NewImgDB(Api, conf.ImgFolder, "dog")
	//CatImg = NewImgDB(Api, conf.ImgFolder, "cat")
	//ParrotImg = NewImgDB(Api, conf.ImgFolder, "parrot")

	db, _ = ConnectDB(conf.DbPath)
	defer db.Close()



	Watcher = NewImgWatcher(db, conf.WatcherConf.MinimalAviable, conf.WatcherConf.MaximumUses, conf.WatcherConf.Checktime, conf.Debug)

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
		router.GET("/" + endpoint, ServeImg(ImgDbs[endpoint]))
	}


	router.Run("0.0.0.0:8080")

}