package main

import (
	"github.com/gin-gonic/gin"
	"github.com/samam0234/my-go-gin-study/demo/step02/health"
	"github.com/samam0234/my-go-gin-study/demo/step02/memo"
)

func newEngine() *gin.Engine {
	r := gin.New()
	health.Register(r)
	memo.Register(r, []memo.Item{{ID: 1, Text: "패키지로 나눈 메모"}})
	return r
}

func main() {
	if err := newEngine().Run(":8080"); err != nil {
		panic(err)
	}
}
