// query_cache.go
package graphchain

import (
	"container/list"
	"sync"
)

type QueryCache struct {
	cache   map[string]*QueryResult
	order   *list.List
	mutex   sync.RWMutex
	maxSize int
}

func NewQueryCache(maxSize int) *QueryCache {
	return &QueryCache{
		cache:   make(map[string]*QueryResult),
		order:   list.New(),
		maxSize: maxSize,
	}
}

func (qc *QueryCache) Get(query string) *QueryResult {
	qc.mutex.RLock()
	defer qc.mutex.RUnlock()

	if result, exists := qc.cache[query]; exists {
		// Move to front (LRU)
		qc.moveToFront(query)
		return result
	}
	return nil
}

func (qc *QueryCache) Set(query string, result *QueryResult) {
	qc.mutex.Lock()
	defer qc.mutex.Unlock()

	if len(qc.cache) >= qc.maxSize {
		qc.evictLRU()
	}

	qc.cache[query] = result
	qc.order.PushFront(query)
}

func (qc *QueryCache) moveToFront(query string) {
	for e := qc.order.Front(); e != nil; e = e.Next() {
		if e.Value.(string) == query {
			qc.order.MoveToFront(e)
			break
		}
	}
}

func (qc *QueryCache) evictLRU() {
	if qc.order.Len() > 0 {
		oldest := qc.order.Back()
		qc.order.Remove(oldest)
		delete(qc.cache, oldest.Value.(string))
	}
}
