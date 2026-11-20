package httpapi

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const requestIDKey = "requestID"

func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.GetHeader("X-Request-Id"))
		id = strings.ReplaceAll(id, "\n", "")
		id = strings.ReplaceAll(id, "\r", "")
		if id == "" {
			id = newID()
		}
		if len(id) > 80 {
			id = id[:80]
		}
		c.Set(requestIDKey, id)
		c.Header("X-Request-Id", id)
		c.Next()
	}
}

func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Next()
	}
}

func accessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Printf("rid=%s %s %s %d %s", c.GetString(requestIDKey), c.Request.Method, c.Request.URL.Path, c.Writer.Status(), time.Since(start))
	}
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

type window struct {
	n     int
	reset time.Time
}

type limiter struct {
	mu sync.Mutex
	m  map[string]*window
}

func newLimiter() *limiter { return &limiter{m: map[string]*window{}} }

func (l *limiter) middleware(limit int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if limit <= 0 || c.Request.Method == http.MethodOptions || c.Request.URL.Path == "/health" {
			c.Next()
			return
		}
		if !l.allow(c.ClientIP(), limit, time.Now()) {
			writeError(c, http.StatusTooManyRequests, "RATE_LIMITED", "요청이 너무 많습니다. 잠시 후 다시 시도하세요.", nil)
			return
		}
		c.Next()
	}
}

func (l *limiter) allow(key string, limit int, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	item := l.m[key]
	if item == nil || now.After(item.reset) {
		l.m[key] = &window{n: 1, reset: now.Add(time.Minute)}
		return true
	}
	if item.n >= limit {
		return false
	}
	item.n++
	return true
}

type savedResponse struct {
	status   int
	body     []byte
	location string
	exp      time.Time
}

type idemStore struct {
	mu sync.Mutex
	m  map[string]savedResponse
}

func newIdem() *idemStore { return &idemStore{m: map[string]savedResponse{}} }

func (s *idemStore) get(userID int64, key string, now time.Time) (savedResponse, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.m[idemKey(userID, key)]
	if !ok || now.After(item.exp) {
		delete(s.m, idemKey(userID, key))
		return savedResponse{}, false
	}
	return item, true
}

func (s *idemStore) put(userID int64, key string, status int, body []byte, location string, exp time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.m) > 1000 {
		now := time.Now()
		for k, v := range s.m {
			if now.After(v.exp) {
				delete(s.m, k)
			}
		}
	}
	copied := make([]byte, len(body))
	copy(copied, body)
	s.m[idemKey(userID, key)] = savedResponse{status: status, body: copied, location: location, exp: exp}
}

func idemKey(userID int64, key string) string {
	return fmt.Sprintf("%d:%s", userID, key)
}
