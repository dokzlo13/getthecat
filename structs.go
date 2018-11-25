package main


type ImgInfo struct {
	ID string `json:"id" gorm:"primary_key"`

	Type string `json:"type"`
	Uses int `gorm:"index" json:"uses"`

	Path string
	Checksum string
	Origin string
	Width int
	Height int
}


type Searhcer interface {
	SearchImages(query string) ([]ImgInfo, error)
}
