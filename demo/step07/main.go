package main

import (
	"fmt"
	"time"

	"github.com/samam0234/my-go-gin-study/internal/auth"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
	fmt.Println("jwt ok")
}

func run() error {
	hashed, err := auth.Hash("password123")
	if err != nil {
		return err
	}
	if !auth.Check(hashed, "password123") || auth.Check(hashed, "nope-nope") {
		return fmt.Errorf("비밀번호 검사 실패")
	}
	token, err := auth.Sign("test-secret-test-secret", 7, time.Hour, time.Now())
	if err != nil {
		return err
	}
	id, err := auth.Parse("test-secret-test-secret", token)
	if err != nil {
		return err
	}
	if id != 7 {
		return fmt.Errorf("id=%d", id)
	}
	return nil
}
