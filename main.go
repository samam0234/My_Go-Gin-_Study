package main

import (
	"log"

	"github.com/samam0234/my-go-gin-study/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
