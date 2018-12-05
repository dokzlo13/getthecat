package main

import (
	"errors"
	"github.com/go-redis/redis"
	log "github.com/sirupsen/logrus"
	"github.com/vmihailenco/msgpack"
	"strconv"
)

type RedisCache struct {
	client *redis.Client
}

type Cache interface {
	Set(prefix string, item ImgInfo) error
	GetRandomId(prefix string) (string, error)
	GetActualId(prefix string) (string, error)
	GetById(prefix string, id string, increment bool) (ImgInfo, error)
	GetScore(prefix string, id string) (float64, error)
	GetAllIds(prefix string) ([]string, error)
	GetIdsInRange(prefix string, min int, max int) ([]string, error)
	Flush() error
}

func NewRedisCache(addr string, db int) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // no password set
		DB:       db, // use default DB
	})
	_, err := client.Ping().Result()
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	cache := &RedisCache{client: client}
	if err := cache.Flush(); err != nil {
		return nil, err
	}
	return cache, nil
}

func (c RedisCache) Set(prefix string, item ImgInfo) error {
	b, err := msgpack.Marshal(&item)
	if err != nil {
		log.Infof("[RedisCache] Error marshalling item %s in cache", item.ID)
		return err
	}

	err = c.client.HSet(prefix, item.ID, b).Err()
	if err != nil {
		log.Infof("[RedisCache] Error saving item %s in cache", item.ID)
	}

	err = c.client.ZAdd(prefix+"_index", redis.Z{Member: item.ID, Score: float64(item.Uses)}).Err()
	if err != nil {
		log.Infof("[RedisCache] Error saving index %s in cache", item.ID)
	}
	return err

}

func (c RedisCache) GetActualId(prefix string) (string, error) {
	var err error

	val, err := c.client.ZRangeByScore(prefix+"_index", redis.ZRangeBy{
		Min:    "-inf",
		Max:    "+inf",
		Offset: 0,
		Count:  1,
	}).Result()

	if err != nil {
		return "", err
	}

	if len(val) < 1 {
		log.Infoln("[RedisCache] Empty index results")
		return "", errors.New("empty set")
	}
	return val[0], nil
}

func (c RedisCache) GetAllIds(prefix string) ([]string, error) {
	return c.client.HKeys(prefix).Result()
}

func (c RedisCache) GetIdsInRange(prefix string, min int, max int) ([]string, error) {
	items, err := c.client.ZRangeByScoreWithScores(prefix+"_index", redis.ZRangeBy{Min: strconv.Itoa(min), Max: strconv.Itoa(min)}).Result()
	if err != nil {
		return []string{}, err
	}
	var data = make([]string, len(items))
	for i, itm := range items {
		data[i] = itm.Member.(string)
	}
	return data, nil
}

func (c RedisCache) GetById(prefix string, id string, increment bool) (ImgInfo, error) {
	var item ImgInfo
	var err error

	itemdata, err := c.client.HGet(prefix, id).Result()
	if err != nil {
		log.Infof("[RedisCache] Error collecting item %s from cache", item.ID)
		return ImgInfo{}, err
	}

	if itemdata == "" {
		log.Infoln("[RedisCache] Empty data results")
		return ImgInfo{}, errors.New("empty set")
	}

	err = msgpack.Unmarshal([]byte(itemdata), &item)
	if err != nil {
		log.Infof("[RedisCache] Error unmarshalling %x... from cache", itemdata[:10])
		return ImgInfo{}, err
	}

	wacthes, err := c.client.ZScore(prefix+"_index", id).Result()
	if err != nil {
		log.Infof("[RedisCache] Error collecting score for %s from cache", id)
		return ImgInfo{}, err
	}
	item.Uses = int(wacthes)

	if increment {
		err = c.client.ZIncrBy(prefix+"_index", 1, id).Err()
		if err != nil {
			log.Infof("[RedisCache] Error incrementing index %s in cache", item.ID)
			return ImgInfo{}, err
		}
	}
	log.Tracef("[RedisCache] Item unmarshalled from cache %s", item.ID)
	return item, nil
}

func (c RedisCache) GetScore(prefix string, id string) (float64, error) {
	return c.client.ZScore(prefix+"_index", id).Result()
}

func (c RedisCache) Flush() error {
	return c.client.FlushDB().Err()
}

func (c RedisCache) GetRandomId(prefix string) (string, error) {
	val, err := c.client.ZRangeByScore(prefix+"_index", redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()
	if err != nil {
		return "", err
	}
	if len(val) < 1 {
		return "", errors.New("empty set")
	}

	rnd := randrange(1, len(val)+1)
	log.Println(rnd)
	val, err = c.client.ZRange(prefix+"_index", int64(rnd)-1, int64(rnd)-1).Result()
	if err != nil {
		return "", err
	}
	if len(val) < 1 {
		return "", errors.New("empty set")
	}

	return val[0], nil
}
