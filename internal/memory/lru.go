package memory

import (
	"container/list"
	"sync"
)

type entry struct {
	key   string
	value []byte
}

type LRUCache struct {
	size  int
	list  *list.List
	table map[string]*list.Element
	mu    sync.Mutex
}

func NewLRU(size int) *LRUCache {
	return &LRUCache{
		size:  size,
		list:  list.New(),
		table: make(map[string]*list.Element),
	}
}

func (c *LRUCache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.table[key]; ok {
		c.list.MoveToFront(el)
		return el.Value.(*entry).value, true
	}
	return nil, false
}

func (c *LRUCache) Set(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.table[key]; ok {
		c.list.MoveToFront(el)
		el.Value.(*entry).value = value
		return
	}
	if c.list.Len() >= c.size {
		if oldest := c.list.Back(); oldest != nil {
			c.list.Remove(oldest)
			delete(c.table, oldest.Value.(*entry).key)
		}
	}
	e := &entry{key: key, value: value}
	el := c.list.PushFront(e)
	c.table[key] = el
}
