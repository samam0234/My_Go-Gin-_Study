package main

import (
	"path/filepath"
	"testing"
)

func TestSQLiteCRUD(t *testing.T) {
	if err := run(filepath.Join(t.TempDir(), "t.db")); err != nil {
		t.Fatal(err)
	}
}
