package main

import (
"fmt"
"sync"
)

const MaxInt = int(^uint(0) >> 1)


type MemCache struct {
	indexlock sync.RWMutex
	index map[string]map[string]int

	valueslock sync.RWMutex
	values map[string]map[string]ImgInfo
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


func (m *MemCache) Set (prefix string, item ImgInfo) error {
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

func (m *MemCache) GetAviable (prefix string, increment bool) (ImgInfo, error) {
	min :=  MaxInt
	var item string
	m.indexlock.Lock()
	for k, v := range m.index[prefix] {
		if v <= min {
			min = v
			item = k
		}
	}
	if increment {
		m.index[prefix][item] ++
	}
	m.indexlock.Unlock()

	if item == "" {
		return ImgInfo{}, fmt.Errorf("Empty set")
	}

	m.valueslock.RLock()
	val, ok := m.values[prefix][item]
	m.valueslock.RUnlock()

	if !ok {
		return ImgInfo{}, fmt.Errorf("Item not found")
	}
	return val, nil
}

func (m *MemCache) GetAllIds (prefix string) ([]string, error) {
	var items []string
	m.indexlock.RLock()
	for k := range m.index[prefix] {
		items = append(items, k)
	}
	m.indexlock.RUnlock()
	return items, nil
}

func (m *MemCache) GetById (prefix string, id string, increment bool) (ImgInfo, error) {
	if increment {
		m.indexlock.Lock()
		m.index[prefix][id] ++
		m.indexlock.Unlock()
	}
	m.valueslock.RLock()
	val, ok := m.values[prefix][id]
	m.valueslock.RUnlock()

	if !ok {
		return ImgInfo{}, fmt.Errorf("Item not found")
	}
	return val, nil
}

func (m *MemCache) GetScore(prefix string, id string) (float64, error) {
	m.indexlock.RLock()
	val, ok := m.index[prefix][id]
	m.indexlock.RUnlock()

	if !ok {
		return 0, fmt.Errorf("Score not found")
	}

	return float64(val), nil
}

func (m *MemCache) GetIdsInRange (prefix string, min int, max int) ([]string, error) {
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
	m.values =  make(map[string]map[string]ImgInfo)
	m.valueslock.Unlock()
	return nil
}