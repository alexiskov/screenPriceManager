package database

import (
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type (
	Storage struct {
		*gorm.DB
	}
)

func Init() (*Storage, error) {
	db, err := gorm.Open(sqlite.Open("base.db"), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	return &Storage{db}, nil
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&Room{}, &Slot{}, &User{})
}
