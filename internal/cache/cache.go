package cache

import (
	"sync"
	"time"
)

type item struct {
	body []byte
	exp  time.Time
}

// TTL 은 짧은 시간 동안 같은 조회 결과를 메모리에 둔다.
type TTL struct {
	mu    sync.Mutex
	items map[string]item
}

func New() *TTL {
	return &TTL{items: map[string]item{}}
}

func (c *TTL) Get(key string, now time.Time) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	it, ok := c.items[key]
	if !ok || now.After(it.exp) {
		delete(c.items, key)
		return nil, false
	}
	out := make([]byte, len(it.body))
	copy(out, it.body)
	return out, true
}

func (c *TTL) Set(key string, body []byte, now time.Time, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	stored := make([]byte, len(body))
	copy(stored, body)
	c.items[key] = item{body: stored, exp: now.Add(ttl)}
}

func (c *TTL) DeletePrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.items {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.items, key)
		}
	}
}
