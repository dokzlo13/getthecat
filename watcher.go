package main

import (
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"time"
)


type ImgWatcher struct {
	DB *gorm.DB
	renew int
	MinimalAviable int
	MaximalUses int
	CollectingMode string
	Cache *Cache
	ImgDBs map[string]ImgDB
}

func initCache(cache *Cache, db *gorm.DB, prefix string) {
	var items []ImgInfo
	db.Model(&ImgInfo{}).Find(&items)
	log.Warningf("Initalizing cache for %s from db...", prefix)
	for _, item := range items {
		cache.Set(prefix, item)
	}
	log.Warningln("Cache initalized!")
}

func checkRemoveEmptyImages(DB *gorm.DB, prefix string) {
	var count int
	DB.Where("type = ? AND (path = '' OR filesize = '0')", prefix).Delete(&ImgInfo{}).Count(&count)
	log.Warningf("[Watcher] Removed values with empty filepaths from DB for prefix \"%s\"", prefix)
}


func updateDB(DB *gorm.DB, items []ImgInfo) func() error {
	return func() error {
		tx := DB.Begin()
		if tx.Error != nil {
			log.Debugf("[Watcher] Database transaction failed with \"%v\", retrying...", tx.Error)
			return tx.Error
		}
		defer tx.Commit()
		for _, img := range items {
			err := tx.Create(&img).Error
			if err != nil {
				tx.Rollback()
				log.Debugf("Database updating failed with \"%v\", retrying...", tx.Error)
				return err
			}
		}
		log.Debugf("[Watcher] Commited %d items to db", len(items))
		return nil
	}
}

func (ag *ImgWatcher) WatchImages(ImgDB ImgDB) {
	log.Warningf("[Watcher] Watcher started for prefix \"%s\"", ImgDB.Prefix)
	ag.ImgDBs[ImgDB.Prefix] = ImgDB

	if ag.CollectingMode != "urls"{
		checkRemoveEmptyImages(ag.DB, ImgDB.Prefix)
	}

	if ag.Cache != nil {
		initCache(ag.Cache, ag.DB, ImgDB.Prefix)
	}

	var collector func(amount int) ([]ImgInfo, error)
	switch ag.CollectingMode {
	case "urls":
		collector = ImgDB.NewUrls
	case "files":
		collector = ImgDB.NewImgs
	default:
		log.Fatalf("[Watcher] found unknown collection mode \"%s\"", ag.CollectingMode)
	}

	var cacheUpdater = func([]ImgInfo) {}
	switch ag.Cache {
	case nil:
		cacheUpdater = func([]ImgInfo) {log.Debugf("No cache backend specified, skipping...")}
	default:
		cacheUpdater = func(items []ImgInfo) {
			log.Debugln("[Watcher] updating cache...")
			for _, img := range items {
				ag.Cache.Set(ImgDB.Prefix, img)
			}
			log.Debugln("[Watcher] Cache updated!")
		}
	}

	for {
		var count int
		ag.DB.Model(&ImgInfo{}).Where("uses < ? AND type = ?", ag.MaximalUses, ImgDB.Prefix).Count(&count)
		log.Debugf("[Watcher] Explored %d aviable images local for prefix \"%s\"", count, ImgDB.Prefix)

		if count < ag.MinimalAviable {
			log.Debugf("[Watcher] DB Watcher detect %d aviable items of expected %d for prefix \"%s\" starting collection task", count, ag.MinimalAviable, ImgDB.Prefix)

			var err error
			collected, err := collector(ag.MinimalAviable - count)

			if err != nil {
				log.Warningf("[Watcher] Error collecting images: \"%s\"", err)
			} else {
				items := make([]ImgInfo, len(collected))
				log.Debugf("[Watcher] DB recieve %d new items from ImgParser", len(items))
				for idx, img := range collected{
					items[idx] = img
					items[idx].Uses = 0
					items[idx].Type = ImgDB.Prefix
				}

				go cacheUpdater(items)

				go retry(10, time.Millisecond*10, updateDB(ag.DB, items))

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


func GetFromDB(DB *gorm.DB, prefix string) (ImgInfo, error) {
	var img ImgInfo
	tx := DB.Begin()
	if tx.Error == nil {
		defer tx.Commit()
		tx.Model(&ImgInfo{}).Where("type = ?", prefix).Order("uses ASC").First(&img)
	} else {
		log.Errorf("Database reading failed with \"%v\"", tx.Error)
		return ImgInfo{}, tx.Error
	}
	return img, nil
}

func GetFromDbById(DB *gorm.DB, prefix string, id string) (ImgInfo, error) {
	var img ImgInfo
	tx := DB.Begin()
	if tx.Error == nil {
		defer tx.Commit()
		tx.Model(&ImgInfo{}).Where("id = AND type = ?", id, prefix).Order("uses ASC").First(&img)
	} else {
		log.Errorf("Database reading failed with \"%v\"", tx.Error)
		return ImgInfo{}, tx.Error
	}
	return img, nil
}

func (ag ImgWatcher) GetImg(prefix string, incrUses bool) (ImgInfo, error) {
	var img ImgInfo
	var err error

	if ag.Cache != nil {
		img, err = ag.Cache.GetAviable(prefix, incrUses)
		//Retrying with DB request
		if err != nil {
			img, err = GetFromDB(ag.DB, prefix)
			if img.ID == "" {
				//Break here, if nothing found
				return ImgInfo{}, fmt.Errorf("No aviable images")
			} else {
				log.Debugf("Cache updated from DB with result %v", ag.Cache.Set(prefix, img))
			}
		}
	}
	//Do it last time
	if img.ID == "" {
		img, err = GetFromDB(ag.DB, prefix)
	}
	//Check last attempt
	if img.ID == "" {
		return ImgInfo{}, fmt.Errorf("No aviable images")
	}

	if !incrUses {
		return img, nil
	}
	go retry(20, time.Millisecond*10,  func() error {
		tx := ag.DB.Begin()
		if tx.Error != nil {
			log.Debugf("Database transaction failed with \"%v\", retrying...", tx.Error)
			return tx.Error
		}
		defer tx.Commit()
		err = tx.Model(&ImgInfo{}).Exec("UPDATE img_infos SET uses = uses + 1 WHERE id = ?", img.ID).Error
		if err != nil {
			log.Debugf("Database updating failed with \"%v\", retrying...", tx.Error)
			return err
		}
		log.Debugf("[Watcher] Incremented uses for item %s to db", img.ID)
		return nil
	})
	return img, nil
}

func (ag ImgWatcher) GetImgById(prefix string, id string, incrUses bool) (ImgInfo, error) {
	var img ImgInfo
	var err error

	if ag.Cache != nil {
		img, err = ag.Cache.GetById(prefix, id, incrUses)
		//Retrying with DB request
		if err != nil {
			img, err = GetFromDbById(ag.DB, prefix, id)
			if img.ID == "" {
				//Break here, if nothing found
				return ImgInfo{}, fmt.Errorf("No aviable images")
			} else {
				log.Debugf("Cache updated from DB with result %v", ag.Cache.Set(prefix, img))
			}
		}
	}
	//Do it last time
	if img.ID == "" {
		img, err = GetFromDbById(ag.DB, prefix, id)
	}
	//Check last attempt
	if img.ID == "" {
		return ImgInfo{}, fmt.Errorf("No aviable images")
	}

	if !incrUses {
		return img, nil
	}
	go retry(20, time.Millisecond*10,  func() error {
		tx := ag.DB.Begin()
		if tx.Error != nil {
			log.Debugf("Database transaction failed with \"%v\", retrying...", tx.Error)
			return tx.Error
		}
		defer tx.Commit()
		err = tx.Model(&ImgInfo{}).Exec("UPDATE img_infos SET uses = uses + 1 WHERE id = ?", img.ID).Error
		if err != nil {
			log.Debugf("Database updating failed with \"%v\", retrying...", tx.Error)
			return err
		}
		log.Debugf("[Watcher] Incremented uses for item %s to db", img.ID)
		return nil
	})
	return img, nil
}


func (ag ImgWatcher) GetFile(prefix string, id string) (*os.File, error) {
	return ag.ImgDBs[prefix].GetImage(id)
}

func NewImgWatcher(db *gorm.DB, conf WatcherConf, debug int) ImgWatcher {
	if  debug == 3 {
		db = db.Debug()
		//db.SetLogger(log.StandardLogger())
	}

	var cache *Cache
	var err error
	if conf.Cache.Addr == "" && conf.Cache.RedisDb == 0 {
		cache = nil
	} else {
		cache, err = NewCache(conf.Cache.Addr, conf.Cache.RedisDb)
	}

	if err != nil {
		log.Errorf("Failed initalizing cache, cache disabled")
		cache = nil
	} else {
		log.Warningf("Cache initalized!")
	}

	return ImgWatcher{DB: db,
					  MinimalAviable:conf.MinimalAviable,
	  				  MaximalUses:conf.MaximumUses,
					  renew:conf.Checktime,
					  CollectingMode:conf.CollectingMode,
	 				  Cache:cache,
					  ImgDBs: map[string]ImgDB{}}
}


func ConnectDB(path string) (*gorm.DB, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {

		err = os.MkdirAll(filepath.Dir(path), os.ModePerm)
		if err != nil {
			log.Fatalf("Error creating database folder for ImgSaver \"%s\"", path)
			return nil, err
		}

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