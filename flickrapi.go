package main

import (
	"encoding/json"
	"github.com/azer/go-flickr"
	log "github.com/sirupsen/logrus"
	"strconv"
	"sync"
	"time"
)

type  Photo  struct {
	ID       string `json:"id"`
	Owner    string `json:"owner"`
	Secret   string `json:"secret"`
	Server   string `json:"server"`
	Farm     int    `json:"farm"`
	Title    string `json:"title"`
	Ispublic int    `json:"ispublic"`
	Isfriend int    `json:"isfriend"`
	Isfamily int    `json:"isfamily"`
}

type FlickrSearchResponse struct {
	Photos struct {
		Page    int    `json:"page"`
		Pages   int    `json:"pages"`
		Perpage int    `json:"perpage"`
		Total   string `json:"total"`
		Photo   []Photo `json:"photo"`
	} `json:"photos"`
	Stat string `json:"stat"`
}

type FlickrImagesResponse struct {
	Sizes struct {
		//Canblog     int `json:"canblog"`
		//Canprint    int `json:"canprint"`
		//Candownload int `json:"candownload"`
		Size        []struct {
			Label  string `json:"label"`
			//Width  int    `json:"width"`
			//Height int    `json:"height"`
			Source string `json:"source"`
			URL    string `json:"url"`
			Media  string `json:"media"`
		} `json:"size"`
	} `json:"sizes"`
	Stat string `json:"stat"`
}


type FlickrApi struct {
	api *flickr.Client
	amount int

}

func NewFlickrApi(key string, amount int) FlickrApi {
	client := &flickr.Client{
		Key: key,
		//Token: "token", // optional
		//Sig: "sig", // optional
	}
	searcher := FlickrApi{api:client, amount:amount}
	return searcher
}

func extractOrigin(client FlickrApi, imginfo Photo, wg *sync.WaitGroup, extracted chan ImgInfo) {
	defer wg.Done()

	var imgsizes []byte
	var err error

	log.Traceln("[Flickr] Requesting", imginfo.ID)
	imgsizes, err = client.api.Request("photos.getSizes", flickr.Params{"photo_id":imginfo.ID})
	if err != nil {
		log.Infoln("[Flickr] Error requesting ", imginfo.ID)
		return
	}
	var imresp FlickrImagesResponse
	err = json.Unmarshal(imgsizes, &imresp)
	if err != nil {
		log.Infoln("[Flickr] Error unmarshalling ", err, string(imgsizes))
		return
	}

	for _, imgsize := range imresp.Sizes.Size {
		if imgsize.Label == "Large" {
			//Here we done
			extracted <-  ImgInfo{Origin:imgsize.Source}
			log.Debugf("[Flickr] Extracted origin %s for \"%s\"", imgsize.Source, imginfo.ID)
			break
		}
	}
	return
}

func (f FlickrApi) SearchImages(query string) ([]ImgInfo, error) {
	log.Debugf("[Flickr] Started searching for \"%s\" ", query)
	page := randrange(0, 100)
	log.Tracef("[Flickr] Requesting results in page \"%d\" ", page)
	d, err := f.api.Request("photos.search",
		 						flickr.Params{
								//"text":query,
								"per_page": strconv.Itoa(f.amount),
								"page": strconv.Itoa(page),
								"tags": query})

	if err != nil {
		return []ImgInfo{}, err
	}

	var resp FlickrSearchResponse
	err = json.Unmarshal(d, &resp)
	if err != nil {
		return []ImgInfo{}, err
	}

	var results []ImgInfo

	wg := new(sync.WaitGroup)
	InfosChan := make(chan ImgInfo)
	wg.Add(len(resp.Photos.Photo))

	for _, img := range resp.Photos.Photo {
		go extractOrigin(f, img, wg, InfosChan)

	}
	go collectImagesInfo(InfosChan, &results)
	wg.Wait()
	time.Sleep(time.Millisecond*50)

	return results, nil
}
