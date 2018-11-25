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
	log.Println("Requesting seacrh for ", query)

	log.Println("\n\n\n THISISNUMBER", strconv.Itoa(randrange(0, 100)), "\n\n\n")


	d, err := f.api.Request("photos.search",
		 						flickr.Params{
								//"text":query,
								"per_page": strconv.Itoa(f.amount),
								"page": strconv.Itoa(randrange(0, 100)),
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

		log.Println("Requesting", pic.ID)
		imgsizes, err = f.api.Request("photos.getSizes", flickr.Params{"photo_id":pic.ID})
		if err != nil {
			log.Println("Error requesting ", pic.ID)
			continue
		}
		var imresp FlickrImagesResponse
		err = json.Unmarshal(imgsizes, &imresp)
		if err != nil {
			log.Println("Error unmarshalling ", err, string(imgsizes))
			continue
		}

		for _, imgsize := range imresp.Sizes.Size {
			if imgsize.Label == "Large" {
				imgs = append(imgs, ImgInfo{Origin:imgsize.Source})
				log.Println("Extracted ", imgsize.Source)
				break
			}
		}

	}
	return imgs, nil
}

//func main() {
//	api := NewFlickrApi("b23bdd338a0bed0ab4ae109b340cb6df")
//	log.Println(api.SearchImages("cat"))
//}