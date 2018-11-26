package main

import (
	log "github.com/sirupsen/logrus"
	"encoding/json"
	"github.com/azer/go-flickr"
	"strconv"
)


type FlickrSearchResponse struct {
	Photos struct {
		Page    int    `json:"page"`
		Pages   int    `json:"pages"`
		Perpage int    `json:"perpage"`
		Total   string `json:"total"`
		Photo   []struct {
			ID       string `json:"id"`
			Owner    string `json:"owner"`
			Secret   string `json:"secret"`
			Server   string `json:"server"`
			Farm     int    `json:"farm"`
			Title    string `json:"title"`
			Ispublic int    `json:"ispublic"`
			Isfriend int    `json:"isfriend"`
			Isfamily int    `json:"isfamily"`
		} `json:"photo"`
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

	var imgs []ImgInfo

	for _, pic := range resp.Photos.Photo {
		var imgsizes []byte

		log.Traceln("[Flickr] Requesting", pic.ID)
		imgsizes, err = f.api.Request("photos.getSizes", flickr.Params{"photo_id":pic.ID})
		if err != nil {
			log.Infoln("[Flickr] Error requesting ", pic.ID)
			continue
		}
		var imresp FlickrImagesResponse
		err = json.Unmarshal(imgsizes, &imresp)
		if err != nil {
			log.Infoln("[Flickr] Error unmarshalling ", err, string(imgsizes))
			continue
		}

		for _, imgsize := range imresp.Sizes.Size {
			if imgsize.Label == "Large" {
				imgs = append(imgs, ImgInfo{Origin:imgsize.Source})
				log.Debugf("[Flickr] Extracted origin %s for \"%s\"", imgsize.Source, pic.ID)
				break
			}
		}
	}
	return imgs, nil
}
