package main

import (
	"path/filepath"
	"testing"
)

func TestFileRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memos.json")
	if err := run(path); err != nil {
		t.Fatal(err)
	}
}
