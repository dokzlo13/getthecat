package main

import (
	"fmt"
	"github.com/micro/go-config"
	"github.com/micro/go-config/source/file"
)

type ServingConf struct {
	Mode    string `json:"mode"`
	ApiPath string `json:"apipath"`
}

type WatcherConf struct {
	MinimalAviable int    `json:"minimal_aviable"`
	MaximumUses    int    `json:"maximal_uses"`
	Checktime      int    `json:"checktime"`
	CollectingMode string `json:"collect"`
	Cache          struct {
		Addr    string `json:"addr"`
		RedisDb int    `json:"db"`
	} `json:"cache"`
}

type Config struct {
	Auth struct {
		ApiKey   string `json:"apikey"`
		GoogleCX string `json:"cx"`
	} `json:"auth"`
	WatcherConf WatcherConf `json:"watcher"`
	ServingConf ServingConf `json:"server"`
	ImgFolder   string      `json:"folder"`
	DbPath      string      `json:"db"`
	Debug       int         `json:"debug"`
	Mode        string      `json:"engine"`
	Endpoints   []string    `json:"endpoints"`
	Logfile     string      `json:"logfile"`
}

func ConfigCheck(conf Config) error {
	if !(conf.WatcherConf.CollectingMode == "urls" || conf.WatcherConf.CollectingMode == "files") {
		return fmt.Errorf("Config invalid argument \"%s\" for \"watcher.mode\"", conf.WatcherConf.CollectingMode)
	}

	if !(conf.ServingConf.Mode == "cache" || conf.ServingConf.Mode == "proxy") {
		return fmt.Errorf("Config invalid argument \"%s\" for \"server.mode\"", conf.ServingConf.Mode)
	}

	if conf.WatcherConf.CollectingMode == "urls" && conf.ServingConf.Mode == "cache" {
		return fmt.Errorf("Config conflict for values \"server.mode: %s\" and \"watcher.collect: %s\"", conf.ServingConf.Mode, conf.WatcherConf.CollectingMode)
	}
	return nil
}

func LoadConfig(path string) (Config, error) {
	var conf Config
	err := config.Load(file.NewSource(
		file.WithPath(path),
	))
	if err != nil {
		return conf, err
	}
	config.Scan(&conf)
	return conf, ConfigCheck(conf)
}
