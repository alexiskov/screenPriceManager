package database

import (
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type (
	Storage struct {
		*gorm.DB
	}
)

func Init() (*Storage, error) {
	db, err := gorm.Open(sqlite.Open("base.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	return &Storage{db}, nil
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&Room{}, &Slot{}, &User{})
}
