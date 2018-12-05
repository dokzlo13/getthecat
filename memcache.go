package main

import (
	"github.com/pkg/errors"
	"sync"
)

const MaxInt = int(^uint(0) >> 1)

type MemCache struct {
	indexlock sync.RWMutex
	index     map[string]map[string]int

	valueslock sync.RWMutex
	values     map[string]map[string]ImgInfo
}

type Cached struct {
	sync.RWMutex
	val map[string]string
}

func NewMemCache() *MemCache {
	return &MemCache{
		//indexlock: *new(sync.RWMutex),
		index: make(map[string]map[string]int),
		//valueslock: *new(sync.RWMutex),
		values: make(map[string]map[string]ImgInfo),
	}
}

func (m *MemCache) Set(prefix string, item ImgInfo) error {
	m.valueslock.Lock()
	if m.values[prefix] == nil {
		m.values[prefix] = map[string]ImgInfo{item.ID: item}
	} else {
		m.values[prefix][item.ID] = item
	}
	//log.Warningln("DAT",m.values)
	m.valueslock.Unlock()

	m.indexlock.Lock()
	if m.index[prefix] == nil {
		m.index[prefix] = map[string]int{item.ID: item.Uses}
	} else {
		m.index[prefix][item.ID] = item.Uses
	}
	//log.Warningln("IDX",m.index)
	m.indexlock.Unlock()
	return nil
}

func (m *MemCache) GetActualId(prefix string) (string, error) {
	min := MaxInt
	var item string
	m.indexlock.Lock()
	for k, v := range m.index[prefix] {
		if v <= min {
			min = v
			item = k
		}
	}
	m.indexlock.Unlock()

	if item == "" {
		return "", errors.New("Empty set")
	}
	return item, nil
}

func (m *MemCache) GetAllIds(prefix string) ([]string, error) {
	var items []string
	m.indexlock.RLock()
	for k := range m.index[prefix] {
		items = append(items, k)
	}
	m.indexlock.RUnlock()
	return items, nil
}

func (m *MemCache) GetRandomId(prefix string) (string, error) {
	m.indexlock.RLock()
	defer m.indexlock.RUnlock()

	rndidx := randrange(1, len(m.index[prefix])+1)
	for k := range m.index[prefix] {
		if rndidx == 1 {
			return k, nil
		}
		rndidx--
	}
	return "", errors.New("empty set")
}

func (m *MemCache) GetById(prefix string, id string, increment bool) (ImgInfo, error) {
	var views int
	m.indexlock.Lock()
	views = m.index[prefix][id]
	if increment {
		m.index[prefix][id]++
	}
	m.indexlock.Unlock()

	m.valueslock.RLock()
	val, ok := m.values[prefix][id]
	m.valueslock.RUnlock()
	val.Uses = views

	if !ok {
		return ImgInfo{}, errors.New("Item not found")
	}
	return val, nil
}

func (m *MemCache) GetScore(prefix string, id string) (float64, error) {
	m.indexlock.RLock()
	val, ok := m.index[prefix][id]
	m.indexlock.RUnlock()

	if !ok {
		return 0, errors.New("Score not found")
	}

	return float64(val), nil
}

func (m *MemCache) GetIdsInRange(prefix string, min int, max int) ([]string, error) {
	var ids []string
	m.indexlock.RLock()
	for id, score := range m.index[prefix] {
		if score >= min && score <= max {
			ids = append(ids, id)
		}
	}
	m.indexlock.RUnlock()
	return ids, nil
}

func (m *MemCache) Flush() error {
	m.indexlock.Lock()
	m.index = make(map[string]map[string]int)
	m.indexlock.Unlock()

	m.valueslock.Lock()
	m.values = make(map[string]map[string]ImgInfo)
	m.valueslock.Unlock()
	return nil
}
