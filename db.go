package main

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"io"
	"log"
	"os"
	"time"
)


type Images struct {
	ID int64 `json:"id" gorm:"primary_key"`
	Type string `json:"type"`
	Path string `json:"path"`
	Uses int `gorm:"index" json:"uses"`
	Checksum string
}

type ImgWatcher struct {
	DB *gorm.DB
	renew int
	MinimalAviable int
	MaximalUses int
}


func md5Hash(filePath string) (string, error) {
	//Initialize variable returnMD5String now in case an error has to be returned
	var returnMD5String string

	//Open the passed argument and check for any error
	file, err := os.Open(filePath)
	if err != nil {
		return returnMD5String, err
	}

	//Tell the program to call the following function when the current function returns
	defer file.Close()

	//Open a new hash interface to write to
	hash := md5.New()

	//Copy the file in the hash interface and check for any error
	if _, err := io.Copy(hash, file); err != nil {
		return returnMD5String, err
	}

	//Get the 16 bytes hash
	hashInBytes := hash.Sum(nil)[:16]

	//Convert the bytes to a string
	returnMD5String = hex.EncodeToString(hashInBytes)

	return returnMD5String, nil

}




func (ag ImgWatcher) WatchImages(ImgDB ImgDB) {
	for {
		var count int

		ag.DB.Model(&Images{}).Where("uses < ? AND type = ?", ag.MaximalUses, ImgDB.Prefix).Count(&count)
		log.Printf("Explored %d aviable images for prefix \"%s\"", count, ImgDB.Prefix)
		if count < ag.MinimalAviable {
			items, err := ImgDB.NewImgs(ag.MinimalAviable - count)
			if err != nil {
				log.Printf("Error collecting images: \"%s\"", err)
			} else {

				checksums := make(map[string]string, len(items))
				for _, imgpath := range items {
					chsm, _ := md5Hash(imgpath)
					checksums[imgpath] = chsm
				}

				tx := ag.DB.Begin()
				for _, imgpath := range items {
					image := Images{Type:ImgDB.Prefix, Path:imgpath, Uses:0, Checksum:checksums[imgpath]}
					tx.Where(Images{Checksum: checksums[imgpath]}).FirstOrCreate(&image)
					tx.Save(&image)
				}
				tx.Commit()
			}
		}
		time.Sleep(time.Second * time.Duration(ag.renew))
	}
}

func (ag ImgWatcher) GetImg(ImgDB ImgDB) string {
	var imag Images
	ag.DB.Where("uses < ? AND type = ?", ag.MaximalUses, ImgDB.Prefix).Order("uses ASC").First(&imag)
	if imag.ID == 0 {
		//If no unused cat - respond used
		ag.DB.Where("uses >= ? AND type = ?", ag.MaximalUses, ImgDB.Prefix).Order("uses ASC").First(&imag)
	}
	if imag.ID == 0 {
		return ""
	}
	imag.Uses ++
	ag.DB.Model(&Images{}).Update(&imag)
	return imag.Path
}

func NewImgWatcher(db *gorm.DB, minimalAviable int,  maximalUses int, checktime int, debug int) ImgWatcher {
	if debug == 1 {
		db = db.Debug()
	}
	return ImgWatcher{DB: db, MinimalAviable:minimalAviable, MaximalUses:maximalUses, renew:checktime}
}


func ConnectDB(path string) (*gorm.DB, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		db, err := gorm.Open("sqlite3", path)
		db.AutoMigrate(&Images{})
		return db, err

	} else {
		return gorm.Open("sqlite3", path)
	}
}