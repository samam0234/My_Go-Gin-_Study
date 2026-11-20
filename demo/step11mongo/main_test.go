package main

import (
	"os"
	"testing"
)

func TestMongoOptional(t *testing.T) {
	if os.Getenv("MONGO_URL") == "" {
		if err := run(""); err != nil {
			t.Fatal(err)
		}
		return
	}
	if err := run(os.Getenv("MONGO_URL")); err != nil {
		t.Fatal(err)
	}
}
