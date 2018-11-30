package main

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)


func ServeImgInfo(prefix string) func(c *gin.Context) {
	return func(c *gin.Context) {
		imginfo, err := Watcher.GetImgById(prefix, c.Param("id"), true)
		if err != nil {
			log.Errorf("No aviable file found with err \"%v\"", err)
			c.AbortWithError(404, err)
			return
		}
		c.JSON(200, imginfo)
	}
}

func ServeRandomImgInfo(prefix string) func(c *gin.Context) {
	return func(c *gin.Context) {
		imginfo, err := Watcher.GetImg(prefix, false)
		if err != nil {
			log.Errorf("No aviable file found with err \"%v\"", err)
			c.AbortWithError(404, err)
			return
		}
		c.JSON(200, imginfo)
	}
}

func ServeImg(prefix string, conf ServingConf) func(c *gin.Context) {
	var responder func(c *gin.Context)
	switch conf.Mode {
	case "cache":
		responder = func(c *gin.Context) {
			imginfo, err := Watcher.GetImgById(prefix, c.Param("id"), true)
			if err != nil {
				log.Errorf("No requested file found with err \"%v\"", err)
				c.AbortWithError(404, err)
				return
			}
			c.File(imginfo.Path)
			return
		}
	case "proxy":
		responder = func(c *gin.Context) {
			imginfo, err := Watcher.GetImgById(prefix, c.Param("id"), true)
			if err != nil {
				log.Errorf("No aviable file found with err \"%v\"", err)
				c.AbortWithError(404, err)
				return
			}
			c.Redirect(303, imginfo.Origin)
		}
	}

	return responder
}

func ServeRandomImg(prefix string, conf ServingConf) func(c *gin.Context) {
	var responder func(c *gin.Context)
	switch conf.Mode {
	case "cache":
		responder = func(c *gin.Context) {
			imginfo, err := Watcher.GetImg(prefix, true)
			if err != nil {
				log.Errorf("No aviable file found with err \"%v\"", err)
				c.AbortWithError(404, err)
				return
			}
			c.File(imginfo.Path)
			return
		}
	case "proxy":
		responder = func(c *gin.Context) {
			imginfo, err := Watcher.GetImg(prefix, true)
			if err != nil {
				log.Errorf("No aviable file found with err \"%v\"", err)
				c.AbortWithError(404, err)
				return
			}
			c.Redirect(303, imginfo.Origin)
			return
		}
	}

	return responder
}
