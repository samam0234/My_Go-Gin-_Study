package memo

import "github.com/gin-gonic/gin"

type Item struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
}

func Register(r *gin.Engine, items []Item) {
	r.GET("/memos", func(c *gin.Context) {
		c.JSON(200, gin.H{"data": items})
	})
}
