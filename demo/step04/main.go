package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type post struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

func newEngine() *gin.Engine {
	posts := []post{{ID: 1, Title: "첫 글"}, {ID: 2, Title: "둘째 글"}}
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/api/posts", func(c *gin.Context) {
		q := c.Query("q")
		out := make([]post, 0)
		for _, item := range posts {
			if q == "" || item.Title == q {
				out = append(out, item)
			}
		}
		c.JSON(http.StatusOK, gin.H{"data": out})
	})
	r.GET("/api/posts/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "id는 숫자여야 합니다."}})
			return
		}
		for _, item := range posts {
			if item.ID == id {
				c.JSON(http.StatusOK, gin.H{"data": item})
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "없습니다."}})
	})
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "경로를 찾을 수 없습니다."}})
	})
	return r
}

func main() {
	if err := newEngine().Run(":8080"); err != nil {
		panic(err)
	}
}
