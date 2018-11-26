package main

import (
	"github.com/micro/go-config"
	"github.com/micro/go-config/source/file"
)

type ServingConf struct {
	Mode     string `json:"mode"`
	Filetype string `json:"filetype"`
}

type WatcherConf struct {
	MinimalAviable int `json:"minimal_aviable"`
	MaximumUses int `json:"maximal_uses"`
	Checktime int `json:"checktime"`
	CollectingMode string `json:"collect"`
}

type Config struct {
	Auth struct {
		ApiKey string `json:"apikey"`
		GoogleCX string `json:"cx"`
	} `json:"auth"`
	WatcherConf WatcherConf `json:"watcher"`
	ServingConf ServingConf `json:"serving"`
	ImgFolder string `json:"folder"`
	DbPath string `json:"db"`
	Debug int `json:"debug"`
	Mode string `json:"engine"`
	Endpoints []string `json:"endpoints"`
	Logfile string `json:"logfile"`
}

func LoadConfig(path string) (Config, error) {
	var conf Config
	err := config.Load(file.NewSource(
		file.WithPath(path),
	))
	if err!=nil {
		return conf, err
	}
	config.Scan(&conf)
	return conf, nil
}