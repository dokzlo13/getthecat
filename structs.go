package main


type ImgInfo struct {
	ID string `json:"id" gorm:"primary_key"`

	Type string `json:"type" gorm:"index"`
	Uses int `gorm:"index" json:"watched"`

	Path string `json:"-"`
	Checksum string
	Origin string
	Width int
	Height int

	Filesize int64 `json:"filesize"`

}


type Searhcer interface {
	SearchImages(query string) ([]ImgInfo, error)
}
