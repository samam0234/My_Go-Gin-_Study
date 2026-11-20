package main

import (
	"context"
	"log"

	"github.com/samam0234/my-go-gin-study/internal/config"
	"github.com/samam0234/my-go-gin-study/internal/mcp"
	"github.com/samam0234/my-go-gin-study/internal/store"
)

func main() {
	cfg := config.Load()
	st, err := store.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(context.Background()); err != nil {
		log.Fatal(err)
	}
	log.Println("MCP stdio 서버를 시작합니다. 도구는 search_posts, get_post 입니다.")
	if err := mcp.ServeStdio(context.Background(), st); err != nil {
		log.Fatal(err)
	}
}
