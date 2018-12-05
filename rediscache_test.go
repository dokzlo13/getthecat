package main

import (
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

var redisaddr = "127.0.0.1:6379"
var redisdb = 4

var items []ImgInfo

func FillItems(lng int) {
	item := make([]ImgInfo, lng)
	for i := range item {
		item[i] = ImgInfo{ID: uuid.NewV4().String()}
	}
	items = item
}

func BenchmarkRedisCache_Set(b *testing.B) {
	b.StopTimer()
	cache, err := NewRedisCache(redisaddr, redisdb)
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

func BenchmarkRedisCache_GetAviable(b *testing.B) {
	b.StopTimer()
	cache, err := NewRedisCache(redisaddr, redisdb)
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
		_, err := cache.GetActualId("bench")
		//log.Println(items[i], err)
		assert.NoError(b, err)
	}
	b.StopTimer()
	cache.client.FlushAll()
}

func TestRedisCache_Set(t *testing.T) {
	cache, err := NewRedisCache(redisaddr, redisdb)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	item := ImgInfo{ID: "f3bc456e-44af-4e52-b9c2-cd88cf1c2c00", Uses: 1, Height: 1, Width: 1, Origin: "test", Filesize: 1, Checksum: "test", Type: "test", Path: "tespath"}
	err = cache.Set("test", item)
	assert.NoError(t, err)
	cache.client.FlushAll()
}

func TestRedisCache_GetActualId(t *testing.T) {
	cache, err := NewRedisCache(redisaddr, redisdb)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	id := "f3bc456e-44af-4e52-b9c2-cd88cf1c2c11"
	wanted := ImgInfo{ID: id, Uses: 1, Height: 1, Width: 1, Origin: "test", Filesize: 1, Checksum: "test", Type: "test", Path: "tespath"}
	cache.Set("test", wanted)
	received, err := cache.GetActualId("test")
	assert.NoError(t, err)
	assert.Equal(t, received, id)
	cache.client.FlushAll()
}

func TestRedisCache_GetById(t *testing.T) {
	cache, err := NewRedisCache(redisaddr, redisdb)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	id := "f3bc456e-44af-4e52-b9c2-cd88cf1c2c22"
	wanted := ImgInfo{ID: id, Uses: 1, Height: 1, Width: 1, Origin: "test", Filesize: 1, Checksum: "test", Type: "test", Path: "tespath"}
	cache.Set("test", wanted)
	received, err := cache.GetById("test", id, true)
	assert.NoError(t, err)
	assert.Equal(t, received, wanted)
	cache.client.FlushAll()
}

func TestRedisCache_NewRedisCache(t *testing.T) {
	cache, err := NewRedisCache(redisaddr, redisdb)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	_, err = cache.GetActualId("test")
	assert.Error(t, err)
	cache.client.FlushAll()
}

func TestRedisCache_GetScore(t *testing.T) {
	cache, err := NewRedisCache(redisaddr, redisdb)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	id := "f3bc456e-44af-4e52-b9c2-cd88cf1c2c22"
	wanted := ImgInfo{ID: id, Uses: 123, Height: 1, Width: 1, Origin: "test", Filesize: 1, Checksum: "test", Type: "test", Path: "tespath"}
	cache.Set("test", wanted)
	received, err := cache.GetById("test", id, true)
	assert.NoError(t, err)
	assert.Equal(t, received, wanted)

	score, err := cache.GetScore("test", id)
	assert.NoError(t, err)
	//Incremented value by GetById
	assert.Equal(t, 123+1, int(score))

	cache.client.FlushAll()

}

func TestRedisCache_GetIdsInRange(t *testing.T) {
	cache, err := NewRedisCache(redisaddr, redisdb)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	FillItems(10)
	for _, itm := range items {
		cache.Set("test", itm)

	}

	total, err := cache.GetIdsInRange("test", 0, 0)
	assert.NoError(t, err)
	assert.Equal(t, 10, len(total))

	cache.client.FlushAll()
}

func TestRedisCache_Flush(t *testing.T) {
	cache, err := NewRedisCache(redisaddr, redisdb)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	id := "f3bc456e-44af-4e52-b9c2-cd88cf1c2c33"
	item := ImgInfo{ID: id, Uses: 1, Height: 1, Width: 1, Origin: "test", Filesize: 1, Checksum: "test", Type: "test", Path: "tespath"}
	err = cache.Set("test", item)
	assert.NoError(t, err)
	_, err = cache.GetById("test", id, false)
	assert.NoError(t, err)

	err = cache.Flush()
	assert.NoError(t, err)

	val, err := cache.GetById("test", id, false)
	assert.Error(t, err)

	assert.NotEqual(t, item, val)
}

func TestRedisCache_GetRandomId(t *testing.T) {
	cache, err := NewRedisCache(redisaddr, redisdb)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	id := "f3bc456e-44af-4e52-b9c2-cd88cf1c2c44"
	item := ImgInfo{ID: id, Uses: 1, Height: 1, Width: 1, Origin: "test", Filesize: 1, Checksum: "test", Type: "test", Path: "tespath"}
	err = cache.Set("test", item)
	assert.NoError(t, err)
	recvdId, err := cache.GetRandomId("test")
	assert.NoError(t, err)
	assert.Equal(t, id, recvdId)
}
