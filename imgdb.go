package main

import (
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

type ImgDB struct {
	Prefix string
	Root string
	Api Searhcer
	saver ImgSaver
}

func NewImgDB (searcher Searhcer, root string, prefix string) ImgDB {
	return ImgDB{Root:root,
				 Prefix:prefix,
				 Api: searcher,
	  			 saver:NewImageSaver(filepath.Join(root, prefix))}
}

func (db ImgDB) NewImgs(amount int) ([]ImgInfo, error) {
	log.Debugf("ImgDB start collecting \"%d\" images for prefix \"%s\"", amount, db.Prefix)
	imgs, err := db.saver.SaveRandomPreparedImage(db.Api, db.Prefix, amount)
	if err != nil {
		return []ImgInfo{}, err
	}
	return imgs, nil
}

func (db ImgDB) LocalImg(id string) (*os.File, error) {
	return db.saver.GetImage(id)
}