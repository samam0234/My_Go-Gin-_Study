package main

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
)

func newEngine() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.GET("/", home)
	r.GET("/ping", ping)
	return r
}

func home(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	name := c.DefaultQuery("name", "Gin")
	if err := page.Execute(c.Writer, map[string]any{"Name": name}); err != nil {
		c.String(http.StatusInternalServerError, "템플릿 오류")
	}
}

func ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
}

func divide(a, b int) (int, error) {
	if b == 0 {
		return 0, fmt.Errorf("0으로 나눌 수 없음")
	}
	return a / b, nil
}

func waitDone() string {
	ch := make(chan string, 1)
	go func() { ch <- "완료" }()
	return <-ch
}

var page = template.Must(template.New("home").Parse(`<!doctype html>
<meta charset="utf-8">
<title>Hello Gin</title>
<h1>안녕하세요, {{.Name}}</h1>
`))

func main() {
	if err := newEngine().Run(":8080"); err != nil {
		panic(err)
	}
}
