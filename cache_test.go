package main

import (
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

var redisaddr = "127.0.0.1:6379"
var redisdb = 4

var items []ImgInfo

func FillItems(lng int) {
	item := make([]ImgInfo, lng)
	for i := range item {
		id, _ := uuid.NewV4()
		item[i] = ImgInfo{ID:id.String()}
	}
	items = item
}

func BenchmarkCache_Set(b *testing.B) {
	logrus.SetLevel(logrus.TraceLevel)
	b.StopTimer()
	cache, err := NewCache(redisaddr, redisdb)
	assert.NoError(b, err)
	if err != nil {
		return
	}
	FillItems(b.N)
	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		err := cache.Set("bench", items[i])
		//log.Println(items[i], err)
		assert.NoError(b, err)
	}
	b.StopTimer()
	cache.client.FlushAll()

}

func BenchmarkCache_GetAviable(b *testing.B) {
	logrus.SetLevel(logrus.TraceLevel)
	b.StopTimer()
	cache, err := NewCache(redisaddr, redisdb)
	assert.NoError(b, err)
	if err != nil {
		return
	}
	FillItems(b.N)

	for i := 0; i < b.N; i++ {
		err := cache.Set("bench", items[i])
		//log.Println(items[i], err)
		assert.NoError(b, err)
	}
	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err := cache.GetAviable("bench", false)
		//log.Println(items[i], err)
		assert.NoError(b, err)
	}
	b.StopTimer()
	cache.client.FlushAll()
}



func TestCache_Set(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	cache, err := NewCache(redisaddr, redisdb)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	item := ImgInfo{ID:"f3bc456e-44af-4e52-b9c2-cd88cf1c2c00", Uses:1, Height:1, Width:1, Origin:"test", Filesize:1, Checksum:"test", Type:"test", Path:"tespath"}
	err = cache.Set("test", item)
	assert.NoError(t, err)
	cache.client.FlushAll()
}

func TestCache_GetAviable(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	cache, err := NewCache(redisaddr, redisdb)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	wanted := ImgInfo{ID:"f3bc456e-44af-4e52-b9c2-cd88cf1c2c11", Uses:1, Height:1, Width:1, Origin:"test", Filesize:1, Checksum:"test", Type:"test", Path:"tespath"}
	cache.Set("test", wanted)
	recieved, err := cache.GetAviable("test", false)
	assert.NoError(t, err)
	assert.Equal(t, recieved, wanted)
	cache.client.FlushAll()
}

func TestCache_GetById(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	cache, err := NewCache(redisaddr, redisdb)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	id := "f3bc456e-44af-4e52-b9c2-cd88cf1c2c22"
	wanted := ImgInfo{ID:id, Uses:1, Height:1, Width:1, Origin:"test", Filesize:1, Checksum:"test", Type:"test", Path:"tespath"}
	cache.Set("test", wanted)
	recieved, err := cache.GetById("test", id, true)
	assert.NoError(t, err)
	assert.Equal(t, recieved, wanted)
	cache.client.FlushAll()
}

func TestCache_NewCache(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	cache, err := NewCache(redisaddr, redisdb)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	_, err = cache.GetAviable("test", false)
	assert.Error(t, err, fmt.Errorf("Empty set"))
	cache.client.FlushAll()
}

func TestCache_GetScore(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)

	cache, err := NewCache(redisaddr, redisdb)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	id := "f3bc456e-44af-4e52-b9c2-cd88cf1c2c22"
	wanted := ImgInfo{ID:id, Uses:123, Height:1, Width:1, Origin:"test", Filesize:1, Checksum:"test", Type:"test", Path:"tespath"}
	cache.Set("test", wanted)
	recieved, err := cache.GetById("test", id, true)
	assert.NoError(t, err)
	assert.Equal(t, recieved, wanted)

	score, err := cache.GetScore("test", id)
	assert.NoError(t, err)
	//Incremented value by GetById
	assert.Equal(t,123 + 1, int(score))


	cache.client.FlushAll()

}
