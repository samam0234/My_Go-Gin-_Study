package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Memo struct {
	Text string `json:"text"`
}

func main() {
	dir, err := os.MkdirTemp("", "step03")
	if err != nil {
		panic(err)
	}
	if err := run(filepath.Join(dir, "memos.json")); err != nil {
		panic(err)
	}
}

func run(path string) error {
	memos := []Memo{{Text: "파일에 남긴 메모"}}
	raw, err := json.MarshalIndent(memos, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return err
	}
	loaded, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var back []Memo
	return json.Unmarshal(loaded, &back)
}
