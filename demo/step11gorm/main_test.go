package main

import (
	"path/filepath"
	"testing"
)

func TestGorm(t *testing.T) {
	if err := run(filepath.Join(t.TempDir(), "n.db")); err != nil {
		t.Fatal(err)
	}
}
