package main

import (
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)


type ImgWatcher struct {
	DB *gorm.DB
	renew int
	MinimalAviable int
	MaximalUses int
	CollectingMode string
}


func (ag ImgWatcher) WatchImages(ImgDB ImgDB) {
	log.Warningf("Watcher task started for prefix \"%s\"", ImgDB.Prefix)
	for {
		var count int
		ag.DB.Model(&ImgInfo{}).Where("uses < ? AND type = ?", ag.MaximalUses, ImgDB.Prefix).Count(&count)
		log.Debugf("[Watcher] Explored %d aviable images local for prefix \"%s\"", count, ImgDB.Prefix)
		if count < ag.MinimalAviable {
			log.Debugf("[Watcher] DB Watcher detect %d aviable items of expected %d for prefix \"%s\" starting collection task", count, ag.MinimalAviable, ImgDB.Prefix)

			var (
				items []ImgInfo
				err error
			)
			switch ag.CollectingMode {
			case "urls":
				items, err = ImgDB.NewUrls(ag.MinimalAviable - count)
			case "files":
				items, err = ImgDB.NewImgs(ag.MinimalAviable - count)
			default:
				log.Fatalf("Watcher found unknown collection mode \"%s\"", ag.CollectingMode)
			}

			log.Debugf("[Watcher] DB recieve %d new items from ImgParser", len(items))

			if err != nil {
				log.Warningf("[Watcher] Error collecting images: \"%s\"", err)
			} else {
				tx := ag.DB.Begin()
				for _, img := range items {
					img.Type = ImgDB.Prefix
					img.Uses = 0
					//tx.Where(ImgInfo{Checksum: img.Checksum}).FirstOrCreate(&img)
					tx.Create(&img)
					//tx.Save(&img)
				}
				tx.Commit()
				log.Debugf("[Watcher] Commited %d items to db", len(items))
			}
		}
		time.Sleep(time.Second * time.Duration(ag.renew))
	}
}


type stop struct {
	error
}

func retry(attempts int, sleep time.Duration, f func() error) error {
	if err := f(); err != nil {
		if s, ok := err.(stop); ok {
			// Return the original error for later checking
			return s.error
		}

		if attempts--; attempts > 0 {
			// Add some randomness to prevent creating a Thundering Herd
			//jitter := time.Duration(rand.Int63n(int64(sleep)))
			//sleep = sleep + jitter/2
			time.Sleep(sleep)
			return retry(attempts, 2*sleep, f)
		}
		return err
	}

	return nil
}


func (ag ImgWatcher) GetImg(ImgDB ImgDB) (ImgInfo, error) {
	var img ImgInfo
	tx := ag.DB.Begin()
	if tx.Error == nil {
		defer tx.Commit()
		tx.Model(&ImgInfo{}).Where("type = ?", ImgDB.Prefix).Order("uses ASC").First(&img)
	} else {
		log.Errorf("Database reading failed with \"%v\"", tx.Error)
		return ImgInfo{}, tx.Error
	}

	if img.ID == "" {
		return ImgInfo{}, fmt.Errorf("No aviable images")
	}

	go retry(20, time.Millisecond,  func() error {
		tx := ag.DB.Begin()
		if tx.Error != nil {
			log.Infof("Database updating failed with \"%v\", retrying...", tx.Error)
			return tx.Error
		}
		defer tx.Commit()
		tx.Model(&ImgInfo{}).Exec("UPDATE img_infos SET uses = uses + 1 WHERE id = ?", img.ID)
		return nil
	})
	return img, nil
}

func (ag ImgWatcher) GetFile(imgDB ImgDB, id string) (*os.File, error) {
	return ImgDB.GetImage(imgDB, id)
}

func NewImgWatcher(db *gorm.DB, conf WatcherConf, debug int) ImgWatcher {
	if  debug == 3 {
		db = db.Debug()
		//db.SetLogger(log.StandardLogger())
	}
	return ImgWatcher{DB: db,
					  MinimalAviable:conf.MinimalAviable,
	  				  MaximalUses:conf.MaximumUses,
					  renew:conf.Checktime,
					  CollectingMode:conf.CollectingMode}
}


func ConnectDB(path string) (*gorm.DB, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		db, err := gorm.Open("sqlite3", path)
		db.AutoMigrate(&ImgInfo{})
		db.Exec("PRAGMA journal_mode=WAL; PRAGMA temp_store = MEMORY; PRAGMA synchronous = OFF;")
		return db, err

	} else {
		db, err := gorm.Open("sqlite3", path)
		db.Exec("PRAGMA journal_mode=WAL; PRAGMA temp_store = MEMORY; PRAGMA synchronous = OFF;")
		return db, err
	}
}