package cache

import (
	"sync"

	"github.com/Ser9unin/ImagePrev/internal/config"
)

type Key string

type Cache interface {
	Set(key string, value interface{}) bool
	Get(key string) (interface{}, bool)
	Clear()
}

type lruCache struct {
	goroutineLock sync.Mutex
	capacity      config.CacheCfg
	queue         List
	items         map[string]*ListItem
}

type cacheItem struct {
	key string
	val interface{}
}

func NewCache(capacity config.CacheCfg) Cache {
	capacityInt := capacity.Capacity
	if capacityInt < 1 {
		capacityInt = 1
	}
	return &lruCache{
		capacity: capacity,
		queue:    NewList(),
		items:    make(map[string]*ListItem, capacityInt),
	}
}

func (l *lruCache) Set(key string, value interface{}) bool {
	l.goroutineLock.Lock()
	defer l.goroutineLock.Unlock()

	_, keyInCache := l.items[key]

	if keyInCache {
		l.queue.Remove(l.items[key])
	} else if l.queue.Len() >= l.capacity.Capacity {
		itemToRemove := l.queue.Back().Value.(cacheItem)

		l.queue.Remove(l.items[itemToRemove.key])
		delete(l.items, itemToRemove.key)
	}

	l.items[key] = l.queue.PushFront(cacheItem{key: key, val: value})

	return keyInCache
}

func (l *lruCache) Get(key string) (interface{}, bool) {
	l.goroutineLock.Lock()
	defer l.goroutineLock.Unlock()

	itemInCache, keyInCache := l.items[key]

	if !keyInCache {
		return nil, false
	}

	itemVal := itemInCache.Value.(cacheItem).val
	l.queue.MoveToFront(itemInCache)

	return itemVal, keyInCache
}

func (l *lruCache) Clear() {
	l.goroutineLock.Lock()
	defer l.goroutineLock.Unlock()

	l.queue = NewList()
	l.items = make(map[string]*ListItem, l.capacity.Capacity)
}
