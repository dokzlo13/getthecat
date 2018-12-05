package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func BenchmarkMemCache_SetCache(b *testing.B) {
	logrus.SetLevel(logrus.TraceLevel)
	b.StopTimer()
	cache := NewMemCache()
	FillItems(b.N)
	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		err := cache.Set("bench", items[i])
		//log.Println(items[i], err)
		assert.NoError(b, err)
	}
	b.StopTimer()
}

func BenchmarkMemCache_GetAviable(b *testing.B) {
	b.StopTimer()
	cache := NewMemCache()
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
}

func TestMemCache_Set(t *testing.T) {
	cache := NewMemCache()
	item := ImgInfo{ID: "f3bc456e-44af-4e52-b9c2-cd88cf1c2c00", Uses: 1, Height: 1, Width: 1, Origin: "test", Filesize: 1, Checksum: "test", Type: "test", Path: "tespath"}
	err := cache.Set("test", item)
	assert.NoError(t, err)
}

func TestMemCache_GetAviable(t *testing.T) {
	cache := NewMemCache()
	id := "f3bc456e-44af-4e52-b9c2-cd88cf1c2c11"
	wanted := ImgInfo{ID: id, Uses: 1, Height: 1, Width: 1, Origin: "test", Filesize: 1, Checksum: "test", Type: "test", Path: "tespath"}
	cache.Set("test", wanted)
	recieved, err := cache.GetActualId("test")
	assert.NoError(t, err)
	assert.Equal(t, id, recieved)
}

func TestMemCache_GetById(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)
	cache := NewMemCache()
	id := "f3bc456e-44af-4e52-b9c2-cd88cf1c2c22"
	wanted := ImgInfo{ID: id, Uses: 1, Height: 1, Width: 1, Origin: "test", Filesize: 1, Checksum: "test", Type: "test", Path: "tespath"}
	cache.Set("test", wanted)
	recieved, err := cache.GetById("test", id, true)
	assert.NoError(t, err)
	assert.Equal(t, recieved, wanted)
}

func TestMemCache_NewMemCache(t *testing.T) {
	cache := NewMemCache()
	_, err := cache.GetActualId("test")
	assert.Error(t, err, fmt.Errorf("Empty set"))
}

func TestMemCache_GetScore(t *testing.T) {
	cache := NewMemCache()
	id := "f3bc456e-44af-4e52-b9c2-cd88cf1c2c22"
	wanted := ImgInfo{ID: id, Uses: 123, Height: 1, Width: 1, Origin: "test", Filesize: 1, Checksum: "test", Type: "test", Path: "tespath"}
	cache.Set("test", wanted)
	recieved, err := cache.GetById("test", id, true)
	assert.NoError(t, err)
	assert.Equal(t, recieved, wanted)
	score, err := cache.GetScore("test", id)
	assert.NoError(t, err)
	//Incremented value by GetById
	assert.Equal(t, 123+1, int(score))
}

func TestMemCache_GetIdsInRange(t *testing.T) {
	cache := NewMemCache()

	FillItems(10)
	for _, itm := range items {
		cache.Set("test", itm)

	}

	total, err := cache.GetIdsInRange("test", 0, 0)
	assert.NoError(t, err)
	assert.Equal(t, 10, len(total))
}

func TestMemCache_Flush(t *testing.T) {
	cache := NewMemCache()
	var err error
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

func TestMemCache_GetRandomId(t *testing.T) {
	cache := NewMemCache()
	var err error
	id := "f3bc456e-44af-4e52-b9c2-cd88cf1c2c33"
	item := ImgInfo{ID: id, Uses: 1, Height: 1, Width: 1, Origin: "test", Filesize: 1, Checksum: "test", Type: "test", Path: "tespath"}
	err = cache.Set("test", item)
	assert.NoError(t, err)

	val, err := cache.GetRandomId("test")
	assert.NoError(t, err)
	assert.Equal(t, id, val)
}
