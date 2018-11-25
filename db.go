package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"os"
	"time"
)


type ImgWatcher struct {
	DB *gorm.DB
	renew int
	MinimalAviable int
	MaximalUses int
}


func (ag ImgWatcher) WatchImages(ImgDB ImgDB) {
	log.Warningf("Watcher task started for prefix \"%s\"", ImgDB.Prefix)
	for {
		var count int
		ag.DB.Model(&ImgInfo{}).Where("uses < ? AND type = ?", ag.MaximalUses, ImgDB.Prefix).Count(&count)
		log.Debugf("Explored %d aviable images local for prefix \"%s\"", count, ImgDB.Prefix)
		if count < ag.MinimalAviable {
			log.Debugf("DB Watcher detect %d aviable items of expected %d for prefix \"%s\" starting collection task", count, ag.MinimalAviable, ImgDB.Prefix)
			items, err := ImgDB.NewImgs(ag.MinimalAviable - count)
			log.Debugf("DB recieve %d new items from ImgParser", len(items))

			if err != nil {
				log.Printf("Error collecting images: \"%s\"", err)
			} else {
				tx := ag.DB.Begin()
				for _, img := range items {
					tx.Where(ImgInfo{Checksum: img.Checksum}).FirstOrCreate(&img)
					img.Type = ImgDB.Prefix
					img.Uses = 0
					tx.Save(&img)
				}
				tx.Commit()
			}
		}
		time.Sleep(time.Second * time.Duration(ag.renew))
	}
}

func (ag ImgWatcher) GetRandomImgPath(ImgDB ImgDB) string {
	var img ImgInfo
	tx := ag.DB.Begin()
	tx.Where("uses < ? AND type = ?", ag.MaximalUses, ImgDB.Prefix).Order("uses ASC").First(&img)
	if img.ID == "" {
		//If no unused cat - respond used
		tx.Where("uses >= ? AND type = ?", ag.MaximalUses, ImgDB.Prefix).Order("uses ASC").First(&img)
	}
	if img.ID == "" {
		return ""
	}
	img.Uses ++
	tx.Model(&ImgInfo{}).Update(&img)
	tx.Commit()
	return img.Path
}

func (ag ImgWatcher) GetRandomImgReader(ImgDB ImgDB) (*os.File, error) {
	var img ImgInfo
	tx := ag.DB.Begin()
	tx.Where("uses < ? AND type = ?", ag.MaximalUses, ImgDB.Prefix).Order("uses ASC").First(&img)
	if img.ID == "" {
		//If no unused cat - respond used
		tx.Where("uses >= ? AND type = ?", ag.MaximalUses, ImgDB.Prefix).Order("uses ASC").First(&img)
	}
	if img.ID == "" {
		return nil, fmt.Errorf("No file with this ID.")
	}
	img.Uses ++
	tx.Model(&ImgInfo{}).Update(&img)
	tx.Commit()
	return ImgDB.LocalImg(img.ID)
}


func NewImgWatcher(db *gorm.DB, minimalAviable int,  maximalUses int, checktime int, debug int) ImgWatcher {
	if debug == 3 {
		db = db.Debug()
		//db.SetLogger(log.StandardLogger())
	}
	return ImgWatcher{DB: db, MinimalAviable:minimalAviable, MaximalUses:maximalUses, renew:checktime}
}


func ConnectDB(path string) (*gorm.DB, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		db, err := gorm.Open("sqlite3", path)
		db.AutoMigrate(&ImgInfo{})
		return db, err

	} else {
		return gorm.Open("sqlite3", path)
	}
}