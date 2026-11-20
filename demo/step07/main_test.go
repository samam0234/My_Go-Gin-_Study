package main

import "testing"

func TestJWT(t *testing.T) {
	if err := run(); err != nil {
		t.Fatal(err)
	}
}
