package main

import (
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type Note struct {
	ID    uint   `gorm:"primaryKey" json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

func main() {
	if err := run(filepath.Join(".", "gorm-demo.db")); err != nil {
		panic(err)
	}
}

func run(path string) error {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	if err := db.AutoMigrate(&Note{}); err != nil {
		return err
	}
	note := Note{Title: "GORM 메모", Body: "태그가 컬럼이 됩니다"}
	if err := db.Create(&note).Error; err != nil {
		return err
	}
	var found Note
	return db.First(&found, note.ID).Error
}
