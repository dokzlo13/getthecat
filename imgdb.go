package main

import (
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

type ImgDB struct {
	Prefix string
	Root string
	Searcher Searhcer
	saver ImgSaver
}

func NewImgDB (searcher Searhcer, root string, prefix string) ImgDB {
	return ImgDB{Root:root,
				 Prefix:prefix,
				 Searcher: searcher,
	  			 saver:NewImageSaver(filepath.Join(root, prefix))}
}

func (db ImgDB) NewImgs(amount int) ([]ImgInfo, error) {
	log.Debugf("[ImgDB] Starting collecting Files for \"%d\" images for prefix \"%s\"", amount, db.Prefix)
	imgs, err := db.saver.GetImagesFiles(db.Searcher, db.Prefix, amount)
	if err != nil {
		return []ImgInfo{}, err
	}
	return imgs, nil
}

func (db ImgDB) NewUrls(amount int) ([]ImgInfo, error) {
	log.Debugf("[ImgDB] Starting collecting URLS-ONLY of \"%d\" images for prefix \"%s\"", amount, db.Prefix)
	imgs, err := db.saver.GetImagesUrls(db.Searcher, db.Prefix, amount)
	if err != nil {
		return []ImgInfo{}, err
	}
	return imgs, nil
}

func (db ImgDB) GetImage(id string) (*os.File, error) {
	//return NewImageSaver(filepath.Join(db.Root, prefix)).GetImage(id)
	return db.saver.GetImage(id)
}