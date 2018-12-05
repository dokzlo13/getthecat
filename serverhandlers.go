package main

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"os"
)

func ServeImgInfo(prefix string) func(c *gin.Context) {
	return func(c *gin.Context) {
		imginfo, err := Watcher.GetImgById(prefix, c.Param("id"), false)
		if err != nil {
			log.Errorf("No aviable file found with err \"%v\"", err)
			c.JSON(404, gin.H{"error": err.Error(), "data": gin.H{}})
			return
		}
		c.JSON(200, gin.H{"error": "", "data": imginfo})
	}
}

func ServeActualImgInfo(prefix string) func(c *gin.Context) {
	return func(c *gin.Context) {
		imginfo, err := Watcher.GetActualImg(prefix, false)
		if err != nil {
			log.Errorf("No aviable file found with err \"%v\"", err)
			c.JSON(404, gin.H{"error": err.Error(), "data": gin.H{}})
			return
		}
		c.JSON(200, gin.H{"error": "", "data": imginfo})
	}
}

func ServeRandomImgInfo(prefix string) func(c *gin.Context) {
	return func(c *gin.Context) {
		imginfo, err := Watcher.GetRandomImg(prefix, false)
		if err != nil {
			log.Errorf("No aviable file found with err \"%v\"", err)
			c.JSON(404, gin.H{"error": err.Error(), "data": gin.H{}})
			return
		}
		c.JSON(200, gin.H{"error": "", "data": imginfo})
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

func ServeActualImg(prefix string, conf ServingConf) func(c *gin.Context) {
	var responder func(c *gin.Context)
	switch conf.Mode {
	case "cache":
		responder = func(c *gin.Context) {
			imginfo, err := Watcher.GetActualImg(prefix, true)
			if err != nil {
				log.Errorf("No aviable file found with err \"%v\"", err)
				c.AbortWithError(404, err)
				return
			}

			f, err := os.Open(imginfo.Path)
			if err != nil {
				c.AbortWithError(404, err)
				return
			}
			defer f.Close()
			c.Header("X-IMAGE-ID", imginfo.ID)
			c.Header("Content-Transfer-Encoding", "binary")
			c.Header("Accept-Ranges", "bytes")
			c.Header("Cache-Control", "max-age=0 no-cache no-store must-revalidate")
			c.DataFromReader(200, imginfo.Filesize, imginfo.Mimetype, f, map[string]string{})
			return
		}
	case "proxy":
		responder = func(c *gin.Context) {
			imginfo, err := Watcher.GetActualImg(prefix, true)
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

func ServeRandomImg(prefix string, conf ServingConf) func(c *gin.Context) {
	var responder func(c *gin.Context)
	switch conf.Mode {
	case "cache":
		responder = func(c *gin.Context) {
			imginfo, err := Watcher.GetRandomImg(prefix, true)
			if err != nil {
				log.Errorf("No aviable file found with err \"%v\"", err)
				c.AbortWithError(404, err)
				return
			}

			f, err := os.Open(imginfo.Path)
			if err != nil {
				c.AbortWithError(404, err)
				return
			}
			defer f.Close()
			c.Header("X-IMAGE-ID", imginfo.ID)
			c.Header("Content-Transfer-Encoding", "binary")
			c.Header("Accept-Ranges", "bytes")
			c.Header("Cache-Control", "max-age=0 no-cache no-store must-revalidate")
			c.DataFromReader(200, imginfo.Filesize, imginfo.Mimetype, f, map[string]string{})
			return
		}
	case "proxy":
		responder = func(c *gin.Context) {
			imginfo, err := Watcher.GetRandomImg(prefix, true)
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
