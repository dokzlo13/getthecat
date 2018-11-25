package main

import (
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/h2non/filetype"
	"github.com/imroc/req"
	"github.com/satori/go.uuid"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type ImgSaver struct {
	 Folder string
}

func NewImageSaver(folder string) ImgSaver {
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		err = os.MkdirAll(folder, os.ModePerm)
		if err != nil {
			log.Fatalf("Error creating images folder for ImgSaver \"%s\"", folder)
		}
	}
	return ImgSaver{Folder:folder}
}

func (i ImgSaver) SaveRandomPreparedImage (searcher Searhcer, query string, amount int) ([]string, error) {
	imgIds, err := i.SaveRandomImages(searcher, query, amount)
	if err != nil {
		return []string{}, err
	}
	err = i.PreprocessImgs(imgIds)
	if err != nil {
		return []string{}, err
	}
	return imgIds, err
}


func (i ImgSaver) SaveRandomImages(searcher Searhcer, query string, amount int) ([]string, error) {
	var err error
	log.Printf("Requesting google for \"%s\"", query)
	data, err := searcher.SearchImages(query)
	if err != nil {
		return []string{}, err
	}
	var res []string
	lng := len(data)
	log.Printf("Request for google is sucessfull! collected:%d items", lng)

	for c:=0; c < amount && c < lng; c++{
		url := data[c]
		log.Print("Collecting img", url, "...")
		r, _ := req.Get(url)
		if err != nil {
			continue
		}
		if !strings.HasPrefix(url, "http") {
			continue
		}
		id, _ := uuid.NewV4()
		err = r.ToFile(filepath.Join(i.Folder, id.String()))
		if err != nil {
			continue
		}
		res = append(res, id.String())
		log.Println("DONE!")
	}
	if len(res) == 0 {
		return []string{}, fmt.Errorf("No aviable images")
	}
	return res, nil
}

func (i ImgSaver) GetImage(id string) (*os.File, error) {
	path := filepath.Join(i.Folder, id)
	//var buf []byte

	buf := make([]byte, 10)
	descr, err := os.OpenFile(path, os.O_RDWR, 0644)
	//descr, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	_, err = descr.Read(buf)
	if err != nil {
		return nil, err
	}

	//Filetype checkout
	kind, unknown := filetype.Match(buf)
	if unknown != nil {
		log.Printf("Unknown: %s", unknown)
		return nil, fmt.Errorf("Unknown file type for \"%s\"!", path)
	}
	if !filetype.IsImage(buf) {
		return nil, fmt.Errorf("Wrong file type for \"%s\" \"%s\"", path, kind.MIME)
	}

	descr.Seek(0, 0)
	return descr, nil

}

func (i ImgSaver)GetFilePath(id string) (string) {
	return filepath.Join(i.Folder, id)
}

func (i ImgSaver)PreprocessImgs(ids []string) error {
	log.Println("Starting preprocessing images!")
	for _, id := range ids {
		descr, err := i.GetImage(id)
		if err != nil {
			return err
		}

		img, err := imaging.Decode(descr)
		if err != nil {
			return err
		}
		dstImage800 := imaging.Resize(img, 800, 0, imaging.Lanczos)
		descr.Seek(0, 0)
		err = imaging.Encode(descr, dstImage800, imaging.PNG)
		if err != nil {
			return err
		}
		log.Printf("Image \"%s\" preprocessed!", id)
	}
	return nil
}