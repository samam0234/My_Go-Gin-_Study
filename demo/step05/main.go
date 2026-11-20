package main

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

type postIn struct {
	Title string `json:"title" binding:"required,min=1,max=200"`
	Body  string `json:"body" binding:"required,min=1,max=2000"`
}

func newEngine() *gin.Engine {
	var mu sync.Mutex
	seen := map[string]int{}
	next := 1
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), allowLocalhost)
	r.POST("/api/posts", func(c *gin.Context) {
		key := c.GetHeader("Idempotency-Key")
		if key != "" {
			mu.Lock()
			if id, ok := seen[key]; ok {
				mu.Unlock()
				c.Header("Idempotency-Replayed", "true")
				c.JSON(http.StatusCreated, gin.H{"data": gin.H{"id": id}})
				return
			}
			mu.Unlock()
		}
		var in postIn
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "입력값을 확인하세요."}})
			return
		}
		mu.Lock()
		id := next
		next++
		if key != "" {
			seen[key] = id
		}
		mu.Unlock()
		c.JSON(http.StatusCreated, gin.H{"data": gin.H{"id": id, "title": in.Title}})
	})
	return r
}

func allowLocalhost(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "http://localhost:5173")
	c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key")
	if c.Request.Method == http.MethodOptions {
		c.Status(http.StatusNoContent)
		return
	}
	c.Next()
}

func main() {
	if err := newEngine().Run(":8080"); err != nil {
		panic(err)
	}
}
