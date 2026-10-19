package database

import "gorm.io/gorm"

type (
	User struct {
		gorm.Model
		Email    string `gorm:"unique;not null"`
		Password string `gorm:"not null"`
		Name     string
		AuthHash string
	}

	Room struct {
		gorm.Model
		Title string
		Slots []Slot `gorm:"foreignKey:RoomID;constraint:OnDelete:CASCADE"`
	}

	Slot struct {
		gorm.Model
		RoomID    uint   `gorm:"index;not null"`
		ImagePath string `gorm:"default:null"`
		Number    uint   `gorm:"not null;default:0"`
	}
)
