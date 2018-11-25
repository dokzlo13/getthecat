package main

import (
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
	for {
		var count int

		ag.DB.Model(&ImgInfo{}).Where("uses < ? AND type = ?", ag.MaximalUses, ImgDB.Prefix).Count(&count)
		log.Printf("Explored %d aviable images for prefix \"%s\"", count, ImgDB.Prefix)
		if count < ag.MinimalAviable {
			items, err := ImgDB.NewImgs(ag.MinimalAviable - count)
			log.Println("\n\nHERE IS IMAGES IN GOROUTINE", items)

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

func (ag ImgWatcher) GetImg(ImgDB ImgDB) string {
	var img ImgInfo
	ag.DB.Where("uses < ? AND type = ?", ag.MaximalUses, ImgDB.Prefix).Order("uses ASC").First(&img)
	if img.ID == "" {
		//If no unused cat - respond used
		ag.DB.Where("uses >= ? AND type = ?", ag.MaximalUses, ImgDB.Prefix).Order("uses ASC").First(&img)
	}
	if img.ID == "" {
		return ""
	}
	img.Uses ++
	ag.DB.Model(&ImgInfo{}).Update(&img)
	return img.Path
}

func NewImgWatcher(db *gorm.DB, minimalAviable int,  maximalUses int, checktime int, debug int) ImgWatcher {
	if debug == 2 {
		db = db.Debug()
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