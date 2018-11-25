package main

import (
	"github.com/micro/go-config"
	"github.com/micro/go-config/source/file"
)


type ServConfig struct {
	Auth struct {
		ApiKey string `json:"apikey"`
		GoogleCX string `json:"cx"`
	} `json:"auth"`
	WatcherConf struct {
		MinimalAviable int `json:"minimal_aviable"`
		MaximumUses int `json:"maximal_uses"`
		Checktime int `json:"checktime"`
	} `json:"watcher"`
	ServingConf struct {
		Mode string `json:"mode"`
		ServingType string  `json:"type"`
	} `json:"serving"`
	ImgFolder string `json:"folder"`
	DbPath string `json:"db"`
	Debug int `json:"debug"`
	Mode string `json:"engine"`
	Endpoints []string `json:"endpoints"`
	Logfile string `json:"logfile"`
}

func LoadConfig(path string) (ServConfig, error) {
	var conf ServConfig
	err := config.Load(file.NewSource(
		file.WithPath(path),
	))
	if err!=nil {
		return conf, err
	}
	config.Scan(&conf)
	return conf, nil
}