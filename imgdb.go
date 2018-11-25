package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
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

func (db ImgDB) NewImgs(amount int) ([]string, error) {
	log.Printf("Starting collecting %d images...", amount)
	ids, err := db.saver.SaveRandomPreparedImage(db.Api, db.Prefix, amount)
	if err != nil {
		return []string{}, err
	}

	var imgs []string
	for _, id := range ids {
		imgs = append(imgs, db.saver.GetFilePath(id))
	}
	return imgs, nil
}

func (db ImgDB) RandLocalImg() (string, error) {
	files, err := ioutil.ReadDir(filepath.Join(db.Root, db.Prefix))
	if err != nil {
		return "", err
	}
	if len(files) < 1 {
		return "", fmt.Errorf("Empty files list")
	}
	filename := files[rand.Intn(len(files))].Name()
	if filename == "" {
		return "", fmt.Errorf("Empty files list")
	}
	return filepath.Join(db.Root, db.Prefix, filename), nil
}

func (db ImgDB) LocalImg(id string) string {
	return db.saver.GetFilePath(id)
}