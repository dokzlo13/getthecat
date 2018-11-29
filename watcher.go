package main

import (
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"sync"
	"time"
)


type ImgWatcher struct {
	DB *gorm.DB
	renew int
	MinimalAviable int
	MaximalUses int
	CollectingMode string
	Cache Cache
	ImgDbsMutex sync.RWMutex
	ImgDBs map[string]ImgDB
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



func (ag ImgWatcher) syncCacheToDb(prefix string) error {
	log.Tracef("[Watcher] Attempting to sync cache for \"%s\" with DB...", prefix)
	knownIds, err := ag.Cache.GetAllIds(prefix)
	states := map[string]int{}
	infos := map[string]ImgInfo{}

	if err != nil {
		return err
	}
	for _, id := range knownIds {
		views, err := ag.Cache.GetScore(prefix, id)
		if err != nil {
			log.Infof("Can not collect rank for %s from cache", id)
		}
		states[id] = int(views)
	}

	for _, id:= range knownIds{
		data, err := ag.Cache.GetById(prefix, id, false)
		if err != nil {
			//continue
		}
		infos[id] = data
	}


	tx := ag.DB.Begin()
	if err:=tx.Error; err != nil {
		log.Tracef("[Watcher] Error creating transaction for sync")
		return err
	}
	for _, id := range knownIds {
		item := infos[id]
		item.Uses = states[id]
		tx.FirstOrCreate(&item, ImgInfo{ID:id})
		tx.Exec("UPDATE img_infos SET uses = ? WHERE id = ? AND type = ?", states[id], id, prefix)
	}
	if err:=tx.Commit().Error; err != nil {
		log.Infof("[Watcher] Error commiting transaction for sync")
		tx.Rollback()
		return err
	}
	log.Tracef("[Watcher] Cache to DB synced!")
	return nil
}

func (ag ImgWatcher) syncDbToCache() {
	var items []ImgInfo
	var err error
	ag.DB.Model(&ImgInfo{}).Find(&items)
	log.Infof("Initalizing cache from db...")
	for _, item := range items {
		err = ag.Cache.Set(item.Type, item)
		if err != nil {
			log.Warningf("Error initalizing cache for %v", item)
		}
	}
	log.Warningln("Cache initalized!")
}


func checkRemoveEmptyImages(DB *gorm.DB, prefix string) {
	var count int
	DB.Where("type = ? AND (path = '' OR filesize = '0')", prefix).Delete(&ImgInfo{}).Count(&count)
	log.Warningf("[Watcher] Removed values with empty filepaths from DB for prefix \"%s\"", prefix)
}

func GetFromDB(DB *gorm.DB, prefix string) (ImgInfo, error) {
	var img ImgInfo
	tx := DB.Begin()
	if err:=tx.Error; err == nil {
		defer tx.Commit()
		tx.Model(&ImgInfo{}).Where("type = ?", prefix).Order("uses ASC").First(&img)
	} else {
		log.Errorf("Database reading failed with \"%v\"", err)
		return ImgInfo{}, tx.Error
	}
	return img, nil
}

func GetFromDbById(DB *gorm.DB, prefix string, id string) (ImgInfo, error) {
	var img ImgInfo
	tx := DB.Begin()
	if err:=tx.Error; err == nil {
		defer tx.Commit()
		tx.Model(&ImgInfo{}).Where("id = AND type = ?", id, prefix).Order("uses ASC").First(&img)
	} else {
		log.Errorf("Database reading failed with \"%v\"", err)
		return ImgInfo{}, tx.Error
	}
	return img, nil
}

func NewImgWatcher(db *gorm.DB, conf WatcherConf, debug int) ImgWatcher {
	if  debug == 3 {
		db = db.Debug()
		//db.SetLogger(log.StandardLogger())
	}

	var cache Cache
	var err error
	if conf.Cache.Addr == "" && conf.Cache.RedisDb == 0 {
		cache = NewMemCache()
		log.Warningln("Using in-memory cache!")
	} else {
		cache, err = NewRedisCache(conf.Cache.Addr, conf.Cache.RedisDb)

		if err != nil {
			log.Errorf("Failed initalizing Redis-cache, cache disabled")
			cache = NewMemCache()
			log.Warningln("Using in-memory cache")

		} else {
			log.Warningf("Using Redis-cache!")
		}
	}


	watcher := ImgWatcher{DB: db,
		MinimalAviable:conf.MinimalAviable,
		MaximalUses:conf.MaximumUses,
		renew:conf.Checktime,
		CollectingMode:conf.CollectingMode,
		Cache:cache,
		ImgDBs: map[string]ImgDB{},
		ImgDbsMutex: *new(sync.RWMutex),
	}

	watcher.syncDbToCache()
	return watcher
}

func (ag ImgWatcher) GetImg(prefix string, incrUses bool) (ImgInfo, error) {
	var img ImgInfo
	var err error

	img, err = ag.Cache.GetAviable(prefix, incrUses)
	//Retrying with DB request
	if err != nil {
		img, err = GetFromDB(ag.DB, prefix)
		if img.ID == "" {
			//Break here, if nothing found
			return ImgInfo{}, fmt.Errorf("No aviable images")
		} else {
			err = ag.Cache.Set(prefix, img)
			if err != nil {
				log.Debugf("[Watcher] Cache from DB updating failed with error %v", err)
			} else {
				log.Debugf("[Watcher] Cache updated from DB!")
			}
		}
	}
	//Check last attempt
	if img.ID == "" {
		return ImgInfo{}, fmt.Errorf("No aviable images")
	}

	if !incrUses {
		return img, nil
	}
	//go retry(20, time.Millisecond*10,  updateItem(ag.DB, img))
	return img, nil
}

func (ag ImgWatcher) GetImgById(prefix string, id string, incrUses bool) (ImgInfo, error) {
	var img ImgInfo
	var err error
	//TODO: Fix Cache getter
	img, err = ag.Cache.GetById(prefix, id, incrUses)
	//Retrying with DB request
	if err != nil {
		img, err = GetFromDbById(ag.DB, prefix, id)
		if img.ID == "" {
			//Break here, if nothing found
			return ImgInfo{}, fmt.Errorf("No aviable images")
		} else {
			log.Debugf("RedisCache updated from DB with result %v", ag.Cache.Set(prefix, img))
		}
	}

	//Check last attempt
	if img.ID == "" {
		return ImgInfo{}, fmt.Errorf("No aviable images")
	}

	if !incrUses {
		return img, nil
	}
	//go retry(20, time.Millisecond*10,  updateItem(ag.DB, img))
	return img, nil
}

func (ag ImgWatcher) GetFile(prefix string, id string) (*os.File, error) {
	ag.ImgDbsMutex.RLock()
	imgdb, ok := ag.ImgDBs[prefix]
	ag.ImgDbsMutex.RUnlock()
	if !ok {
		fmt.Println()
		return nil, fmt.Errorf("ImgDB Locked")
	}
	return imgdb.GetImage(id)
}

func (ag *ImgWatcher) WatchImages(ImgDB ImgDB) {
	log.Warningf("[Watcher] Watcher started for prefix \"%s\"", ImgDB.Prefix)
	ag.ImgDbsMutex.Lock()
	ag.ImgDBs[ImgDB.Prefix] = ImgDB
	ag.ImgDbsMutex.Unlock()

	if ag.CollectingMode != "urls"{
		checkRemoveEmptyImages(ag.DB, ImgDB.Prefix)
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

	cacheUpdater := func(items []ImgInfo) {
		log.Debugln("[Watcher] updating cache...")
		for _, img := range items {
			ag.Cache.Set(ImgDB.Prefix, img)
		}
		log.Debugln("[Watcher] Cache updated!")
		}

	for {
		var count int

		items, err := ag.Cache.GetIdsInRange(ImgDB.Prefix, ag.MaximalUses-1, ag.MaximalUses-1)
		if err != nil {
			log.Fatalf("Error receiving stats from cache \"%v\"", err)
		}
		count = len(items)
		//ag.DB.Model(&ImgInfo{}).Where("uses < ? AND type = ?", ag.MaximalUses, ImgDB.Prefix).Count(&count)
		log.Debugf("[Watcher] Explored %d aviable images local for prefix \"%s\"", count, ImgDB.Prefix)

		if count < ag.MinimalAviable {
			log.Debugf("[Watcher] DB Watcher detect %d aviable items of expected %d for prefix \"%s\" starting collection task", count, ag.MinimalAviable, ImgDB.Prefix)

			var err error
			collected, err := collector(ag.MinimalAviable - count)

			if err != nil {
				log.Warningf("[Watcher] Error collecting images: \"%s\"", err)
			} else {
				items := make([]ImgInfo, len(collected))
				log.Debugf("[Watcher] Collected %d new items from ImgParser for prefix %s", len(items), ImgDB.Prefix)
				for idx, img := range collected{
					items[idx] = img
					items[idx].Uses = 0
					items[idx].Type = ImgDB.Prefix
				}
				cacheUpdater(items)
				//go retry(10, time.Millisecond*10, updateDB(ag.DB, items))
			}

		}
		time.Sleep(time.Second * time.Duration(ag.renew))
		//go retry(10, time.Millisecond * 10, func() error {return ag.syncCacheToDb(ImgDB.Prefix)})
	}
}

func (ag ImgWatcher) Sync() error {
	log.Infoln("[Watcher] Syncing DB...")
	ag.ImgDbsMutex.RLock()
	var err error
	for _, i := range ag.ImgDBs {
		err = ag.syncCacheToDb(i.Prefix)
		if err != nil {
			return err
		}
	}
	ag.ImgDbsMutex.RUnlock()
	log.Infoln("[Watcher] Sync complete")
	return nil
}

func (ag ImgWatcher) StartSync() {
	log.Warningln("[Watcher] DB sync task started!")
	for {
		time.Sleep(time.Second * time.Duration(ag.renew))
		retry(5, time.Millisecond, func() error {return ag.Sync()})
	}
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