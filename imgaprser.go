package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/h2non/filetype"
	"github.com/imroc/req"
	"github.com/satori/go.uuid"
	"log"
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

func (i ImgSaver) SaveRandomPreparedImage (searcher Searhcer, query string, amount int) ([]ImgInfo, error) {
	imgRaw, err := i.SaveRandomImages(searcher, query, amount)
	if err != nil {
		return []ImgInfo{}, err
	}

	log.Println("\n\nHERE IS IMAGES SAVED", imgRaw)
	imgData, err := i.PreprocessImgs(imgRaw)
	if err != nil {
		return []ImgInfo{}, err
	}
	log.Println("\n\nHERE IS IMAGES PREPROCESSED", imgRaw)

	return imgData, err
}


func (i ImgSaver) SaveRandomImages(searcher Searhcer, query string, amount int) ([]ImgInfo, error) {
	var err error
	log.Printf("Creating search for \"%s\"", query)
	data, err := searcher.SearchImages(query)
	if err != nil {
		return []ImgInfo{}, err
	}
	//var res []string
	lng := len(data)
	log.Printf("Request for seacrh is sucessfull! collected:%d items", lng)

	for c:=0; c < amount && c < lng; c++{
		url := data[c].Origin
		log.Print("Collecting img", url, "...")
		r, _ := req.Get(url)
		if err != nil {
			continue
		}
		if !strings.HasPrefix(url, "http") {
			continue
		}
		id, _ := uuid.NewV4()
		path := filepath.Join(i.Folder, id.String())

		err = r.ToFile(path)
		if err != nil {
			continue
		}
		data[c].Path = path
		data[c].ID = id.String()
		log.Println("DONE!")
	}
	if len(data) == 0 {
		return []ImgInfo{}, fmt.Errorf("No aviable images")
	}
	return data, nil
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

func getImageDimension(file *os.File) (int, int) {
	image, _, err := image.DecodeConfig(file)
	if err != nil {
		log.Printf("Error collecting dimensions from image: %v\n", err)
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


func (i ImgSaver)PreprocessImgs(imgs []ImgInfo) ([]ImgInfo, error) {
	log.Println("Starting preprocessing images!")

	for idx, _ := range imgs {
		descr, err := i.GetImage(imgs[idx].ID)
		if err != nil {
			continue
			//return err
		}

		img, err := imaging.Decode(descr)
		if err != nil {
			continue
			//return err
		}
		dstImage800 := imaging.Fit(img, 800, 600, imaging.Lanczos)
		descr.Seek(0, 0)
		err = imaging.Encode(descr, dstImage800, imaging.PNG)
		if err != nil {
			continue
			//return err
		}
		log.Printf("Image \"%s\" preprocessed!", imgs[idx].ID)

		descr.Seek(0, 0)
		imgs[idx].Width, imgs[idx].Height = getImageDimension(descr)

		descr.Seek(0, 0)
		checksum, err := md5Hash(descr)
		if err != nil {
			log.Printf("Error collecting md5 of img %s", imgs[idx].ID)
		} else {
			imgs[idx].Checksum = checksum
		}

		log.Println("IMAG INFOS", imgs[idx])

	}
	log.Println("IMAG INFOS", imgs)

	return imgs, nil
}