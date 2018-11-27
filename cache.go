package main

import (
	"fmt"
	"github.com/go-redis/redis"
	log "github.com/sirupsen/logrus"
	"github.com/vmihailenco/msgpack"

)

type Cache struct {
	client *redis.Client
}

func NewCache (addr string, db int) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // no password set
		DB:       db,  // use default DB
	})
	_, err := client.Ping().Result()
	if err != nil {
		return nil, err
	}
	err = client.FlushAll().Err()
	if err != nil {
		return nil, err
	}
	return &Cache{client:client}, nil
}

func (c Cache) Set (prefix string, item ImgInfo) error {
	b, err := msgpack.Marshal(&item)
	if err != nil {
		log.Infof("[Cache] Error marshalling item %s in cache", item.ID)
		return err
	}

	err = c.client.HSet(prefix, item.ID, b).Err()
	if err != nil {
		log.Infof("[Cache] Error saving item %s in cache", item.ID)
	}

	err = c.client.ZAdd(prefix + "_index", redis.Z{Member:item.ID, Score:float64(item.Uses)}).Err()
	if err != nil {
		log.Infof("[Cache] Error saving index %s in cache", item.ID)
	}
	return err

}

func (c Cache) GetAviable (prefix string, increment bool) (ImgInfo, error) {
	var err error

	val, err := c.client.ZRangeByScore(prefix + "_index", redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
		Offset: 0,
		Count: 1,
	}).Result()

	if err != nil {
		return ImgInfo{}, err
	}

	if len(val) < 1 {
		log.Infoln("[Cache] Empty index results")
		return ImgInfo{}, fmt.Errorf("Empty set")
	}

	//b := []byte(val[0])

	return c.GetById(prefix, val[0], increment)

}


func (c Cache) GetById (prefix string, id string, increment bool) (ImgInfo, error) {
	var item ImgInfo
	var err error

	itemdata, err := c.client.HGet(prefix, id).Result()
	if err != nil {
		log.Infof("[Cache] Error collecting item %s from cache", item.ID)
		return ImgInfo{}, err
	}

	if itemdata == "" {
		log.Infoln("[Cache] Empty data results")
		return ImgInfo{}, fmt.Errorf("Empty set")
	}

	err = msgpack.Unmarshal([]byte(itemdata), &item)
	if err != nil {
		log.Infof("[Cache] Error unmarshalling %x... from cache", itemdata[:10])
		return ImgInfo{}, err
	}
	if increment {
		err = c.client.ZIncrBy(prefix + "_index", 1, id).Err()
		if err != nil {
			log.Infof("[Cache] Error incrementing index %s in cache", item.ID)
			return ImgInfo{}, err
		}
	}
	log.Tracef("[Cache] Item unmarshalled from cache %s", item.ID)
	return item, nil
}

//func ExampleMarshal() {
//	type Item struct {
//		Foo string
//	}
//
//	b, err := msgpack.Marshal(&Item{Foo: "bar"})
//	if err != nil {
//		panic(err)
//	}
//
//	var item Item
//	err = msgpack.Unmarshal(b, &item)
//	if err != nil {
//		panic(err)
//	}
//	fmt.Println(item.Foo)
//	// Output: bar
//}
//
//func main() {
//	client := redis.NewClient(&redis.Options{
//		Addr:     "localhost:6379",
//		Password: "", // no password set
//		DB:       5,  // use default DB
//	})
//
//	pong, err := client.Ping().Result()
//	fmt.Println(pong, err)
//
//	mem := redis.Z{Member:"HELLO"}
//
//	//{
//	//	val, err := client.ZAdd("cats", mem).Result()
//	//	log.Println(err, val)
//	//}
//
//	log.Println(client.ZIncr("cats", mem).Result())
//
//	val, err := client.ZRangeByScore("cats", redis.ZRangeBy{
//		Min: "-inf",
//		Max: "+inf",
//		Offset: 0,
//		Count: 1,
//	}).Result()
//
//
//	log.Println(err, []byte(val[0]), val)
//}