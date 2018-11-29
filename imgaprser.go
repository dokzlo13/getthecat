package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/h2non/filetype"
	"github.com/imroc/req"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	log.Tracef("[ImgSaver] Request for seacrh is successfull! collected:%d items", lng)
	var results []ImgInfo
	for c:=0; c < amount && c < lng; c++{
		url := data[c].Origin
		id, _ := uuid.NewV4()
		data[c].ID = id.String()
		log.Debugf("[ImgSaver] Collecting img ORIGINS %s SUCCEED", url)
		results = append(results, data[c])
	}
	if len(results) == 0 {
		return []ImgInfo{}, fmt.Errorf("No aviable images")
	}
	return results, nil
}

func downloadImage(imginfo ImgInfo, rootpath string, wg *sync.WaitGroup, processed chan ImgInfo) {
	defer wg.Done()
	url := imginfo.Origin
	if url == "" {
		log.Infof("[ImgSaver] Empty URL recieved %s, download FAILED", url)
		return
	}
	request := req.New()
	r, err := request.Get(url)
	if err != nil {
		log.Infof("[ImgSaver] Collecting img %s FAILED", url)
		return
	}
	if !strings.HasPrefix(url, "http") {
		log.Infof("[ImgSaver] Collecting img %s FAILED", url)
		return
	}
	id, _ := uuid.NewV4()
	path := filepath.Join(rootpath, id.String())

	err = r.ToFile(path)
	if err != nil {
		log.Infof("[ImgSaver] Collecting img %s FAILED", url)
		return
	}
	imginfo.Path = path
	imginfo.ID = id.String()
	log.Debugf("[ImgSaver] Collecting img %s SUCEED", url)

	processed <- imginfo
}

func (i ImgSaver) saveRandomImages(searcher Searhcer, query string, amount int) ([]ImgInfo, error) {
	var err error
	log.Tracef("[ImgSaver] Creating search for \"%s\"", query)
	data, err := searcher.SearchImages(query)
	if err != nil {
		return []ImgInfo{}, err
	}
	log.Tracef("[ImgSaver] Request for search is successfull! collected:%d items", len(data))

	var results []ImgInfo
	wg := new(sync.WaitGroup)
	InfosChan := make(chan ImgInfo)
	if len(data) < amount {
		return []ImgInfo{}, fmt.Errorf("imgsaver recieve less items, then amount")
	}
	wg.Add(len(data[:amount]))


	for _, img := range data[:amount] {
		go downloadImage(img, i.Folder, wg, InfosChan)
	}
	go func (){
		wg.Wait()
		close(InfosChan)
	} ()
	collectImagesInfo(InfosChan, &results)

	if len(results) == 0 {
		return []ImgInfo{}, fmt.Errorf("Empty saved images list")
	}

	return results, nil
}

func (i ImgSaver) GetImage(id string) (*os.File, error) {
	path := filepath.Join(i.Folder, id)
	//var buf []byte

	descr, err := os.OpenFile(path, os.O_RDWR, 0644)
	//descr, err := os.Open(path)
	if err != nil {
		log.Infof("[ImgGetter] Error openning file: %s err: %v", path, err)
		return nil, err
	}
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

func preprocessImg(imginfo ImgInfo, descr *os.File, wg *sync.WaitGroup, processed chan ImgInfo) {
	defer wg.Done()
	log.Tracef("[Preprocess] Preprocessing image %s", descr.Name())
	imginfo.Width, imginfo.Height = getImageDimension(descr)
	var err error

	descr.Seek(0, 0)
	buf := make([]byte, 10)
	_, err = descr.Read(buf)
	if err != nil {
		log.Infof("[ImgGetter] Error reading file for fetching mimetype: %s err: %v", imginfo.ID, err)
		return
	}

	//Filetype checkout
	_, err = filetype.Match(buf)
	if err != nil {
		log.Debugf("[ImgGetter] Wrong mimetype for: %s err: %v", imginfo.ID, err)
		return
	}
	if !filetype.IsImage(buf) {
		log.Debugf("[ImgGetter] Wrong mimetype for: %s err: %v", imginfo.ID, err)
		return
	}

	descr.Seek(0, 0)
	if imginfo.Width != 800 || imginfo.Height != 600 {
		descr.Seek(0, 0)
		img, err := imaging.Decode(descr)
		if err != nil {
			log.Infof("[Preprocess] Error opening as image: %s with err \"%v\"", imginfo.ID, err)
			return
		}
		dstImage800 := imaging.Fit(img, 800, 600, imaging.Lanczos)
		descr.Seek(0, 0)
		err = imaging.Encode(descr, dstImage800, imaging.PNG)
		if err != nil {
			log.Infof("[Preprocess] Error saving image: %s with err \"%v\"", imginfo.ID, err)
			return
		}
		descr.Seek(0, 0)
		imginfo.Width, imginfo.Height = getImageDimension(descr)
	}


	descr.Seek(0, 0)
	checksum, err := md5Hash(descr)
	if err != nil {
		log.Infof("[Preprocess] Error collecting md5 of img %s", imginfo.ID)
		return
	} else {
		imginfo.Checksum = checksum
	}

	fi, err := descr.Stat()
	if err != nil {
		log.Infof("[Preprocess] Error collecting file stats %s, %v", imginfo.ID, err)
		return
	} else {
		imginfo.Filesize = fi.Size()
	}

	log.Debugf("[Preprocess] Image %s preprocessed!", imginfo.ID)
	processed <- imginfo
	return
}

func collectImagesInfo(channel chan ImgInfo, InfosList *[]ImgInfo) {
	for s := range channel {
		*InfosList = append(*InfosList, s)
	}
}


func (i ImgSaver)preprocessImgs(imgs []ImgInfo) ([]ImgInfo, error) {
	log.Trace("[Preprocess] Starting preprocessing images!")
	var results []ImgInfo
	wg := new(sync.WaitGroup)
	InfosChan := make(chan ImgInfo)


	for _, img := range imgs {
		descr, err := i.GetImage(img.ID)
		if err != nil {
			log.Infof("[Preprocess] Error fetching image from disc: %s with err \"%v\"", img.ID, err)
			continue
		}
		wg.Add(1)
		go preprocessImg(img, descr, wg, InfosChan)
	}
	go func (){
		wg.Wait()
		close(InfosChan)
	} ()
	collectImagesInfo(InfosChan, &results)
	//time.Sleep(time.Millisecond*50)

	return results, nil
}