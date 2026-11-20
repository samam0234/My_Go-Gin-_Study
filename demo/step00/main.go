package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/joho/godotenv"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	_ = godotenv.Load(".env")
	fmt.Println("go", runtime.Version())
	fmt.Println("module github.com/samam0234/my-go-gin-study")
	fmt.Println("APP_ENV", getenv("APP_ENV", "development"))
	if os.Getenv("JWT_SECRET") == "" {
		fmt.Println("JWT_SECRET 설정됨? 아니오")
	} else {
		fmt.Println("JWT_SECRET 설정됨? 예")
	}
	return nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
