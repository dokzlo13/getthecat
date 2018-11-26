package main

import (
	log "github.com/sirupsen/logrus"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/h2non/filetype"
	"github.com/imroc/req"
	"github.com/satori/go.uuid"
	"image"
	"io"
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

func (i ImgSaver) GetImagesFiles(searcher Searhcer, query string, amount int) ([]ImgInfo, error) {
	imgRaw, err := i.saveRandomImages(searcher, query, amount)
	if err != nil {
		return []ImgInfo{}, err
	}

	log.Tracef("[ImgParser] Saved %d images for query %s", len(imgRaw), query)
	imgData, err := i.preprocessImgs(imgRaw)
	if err != nil {
		return []ImgInfo{}, err
	}
	log.Tracef("[ImgParser] Preprocessed %d images for query %s", len(imgData), query)

	return imgData, err
}

func (i ImgSaver) GetImagesUrls(searcher Searhcer, query string, amount int) ([]ImgInfo, error) {
	var err error
	log.Tracef("[ImgSaver] Creating search for \"%s\"", query)
	data, err := searcher.SearchImages(query)
	if err != nil {
		return []ImgInfo{}, err
	}
	lng := len(data)
	log.Tracef("[ImgSaver] Request for seacrh is sucessfull! collected:%d items", lng)
	var results []ImgInfo
	for c:=0; c < amount && c < lng; c++{
		url := data[c].Origin
		id, _ := uuid.NewV4()
		data[c].ID = id.String()
		log.Debugf("[ImgSaver] Collecting img ORIGINS %s SUCEED", url)
		results = append(results, data[c])
	}
	if len(results) == 0 {
		return []ImgInfo{}, fmt.Errorf("No aviable images")
	}
	return results, nil
}


func (i ImgSaver) saveRandomImages(searcher Searhcer, query string, amount int) ([]ImgInfo, error) {
	var err error
	log.Tracef("[ImgSaver] Creating search for \"%s\"", query)
	data, err := searcher.SearchImages(query)
	if err != nil {
		return []ImgInfo{}, err
	}
	lng := len(data)
	log.Tracef("[ImgSaver] Request for seacrh is sucessfull! collected:%d items", lng)

	var results []ImgInfo
	for c:=0; c < amount && c < lng; c++{
		url := data[c].Origin
		r, _ := req.Get(url)
		if err != nil {
			log.Infof("[ImgSaver] Collecting img %s FAILED", url)
			continue
		}
		if !strings.HasPrefix(url, "http") {
			log.Infof("[ImgSaver] Collecting img %s FAILED", url)
			continue
		}
		id, _ := uuid.NewV4()
		path := filepath.Join(i.Folder, id.String())

		err = r.ToFile(path)
		if err != nil {
			log.Infof("[ImgSaver] Collecting img %s FAILED", url)
			continue
		}
		data[c].Path = path
		data[c].ID = id.String()
		log.Debugf("[ImgSaver] Collecting img %s SUCEED", url)
		results = append(results, data[c])
	}
	if len(results) == 0 {
		return []ImgInfo{}, fmt.Errorf("No aviable images")
	}
	return results, nil
}

func (i ImgSaver) GetImage(id string) (*os.File, error) {
	path := filepath.Join(i.Folder, id)
	//var buf []byte

	buf := make([]byte, 10)
	descr, err := os.OpenFile(path, os.O_RDWR, 0644)
	//descr, err := os.Open(path)
	if err != nil {
		log.Infof("[ImgGetter] Error openning file: %s err: %v", path, err)
		return nil, err
	}
	_, err = descr.Read(buf)
	if err != nil {
		log.Infof("[ImgGetter] Error reading file for fetching mimetype: %s err: %v", path, err)
		return nil, err
	}

	//Filetype checkout
	kind, unknown := filetype.Match(buf)
	if unknown != nil {
		log.Debugf("[ImgGetter] Wrong mimetype for: %s err: %v", path, err)
		return nil, fmt.Errorf("Unknown file type for \"%s\"!", path)
	}
	if !filetype.IsImage(buf) {
		log.Debugf("[ImgGetter] Wrong mimetype for: %s err: %v", path, err)
		return nil, fmt.Errorf("Wrong file type for \"%s\" \"%s\"", path, kind.MIME)
	}

	descr.Seek(0, 0)
	return descr, nil

}


func getImageDimension(file *os.File) (int, int) {
	image, _, err := image.DecodeConfig(file)
	if err != nil {
		log.Tracef("[getImageDimension] Error collecting dimensions from image: %v\n", err)
	}
	return image.Width, image.Height
}


func md5Hash(file *os.File) (string, error) {
	//Initialize variable returnMD5String now in case an error has to be returned
	var returnMD5String string


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


func (i ImgSaver)preprocessImgs(imgs []ImgInfo) ([]ImgInfo, error) {
	log.Trace("[Preprocess] Starting preprocessing images!")

	var results []ImgInfo
	for idx := range imgs {
		descr, err := i.GetImage(imgs[idx].ID)
		if err != nil {
			log.Infof("[Preprocess] Error collecting image: %s with err \"%v\"", imgs[idx].ID, err)
			continue
		}

		img, err := imaging.Decode(descr)
		if err != nil {
			log.Infof("[Preprocess] Error opening as image: %s with err \"%v\"", imgs[idx].ID, err)
			continue
		}
		dstImage800 := imaging.Fit(img, 800, 600, imaging.Lanczos)
		descr.Seek(0, 0)
		err = imaging.Encode(descr, dstImage800, imaging.PNG)
		if err != nil {
			log.Infof("[Preprocess] Error saving image: %s with err \"%v\"", imgs[idx].ID, err)
			continue
			//return err
		}
		descr.Seek(0, 0)
		imgs[idx].Width, imgs[idx].Height = getImageDimension(descr)

		descr.Seek(0, 0)
		checksum, err := md5Hash(descr)
		if err != nil {
			log.Infof("[Preprocess] Error collecting md5 of img %s", imgs[idx].ID)
			continue
		} else {
			imgs[idx].Checksum = checksum
		}

		fi, err := descr.Stat()
		if err != nil {
			log.Infof("[Preprocess] Error collecting file stats %s, %v", img, err)
			continue
		} else {
			imgs[idx].Filesize = fi.Size()
		}

		results = append(results, imgs[idx])
		log.Debugf("[Preprocess] Image \"%s\" preprocessed!", imgs[idx].ID)
	}

	return imgs, nil
}