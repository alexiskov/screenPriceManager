package database

import (
	"context"

	"gorm.io/gorm"
)

type (
	RoomsData struct {
		*gorm.DB
	}

	RoomsMGR interface {
		GetAll(ctx context.Context) ([]Room, error)
		Add(ctx context.Context, room *Room) error
		Delete(ctx context.Context, id uint) (int, error)
	}
)

func (stg Storage) Rooms() RoomsMGR {
	var rmgr RoomsMGR = RoomsData{stg.DB}
	return rmgr
}

func (rd RoomsData) GetAll(ctx context.Context) ([]Room, error) {
	return gorm.G[Room](rd.DB).Find(ctx)
}

func (rd RoomsData) Add(ctx context.Context, room *Room) error {
	return gorm.G[Room](rd.DB).Create(ctx, room)
}

func (rd RoomsData) Delete(ctx context.Context, id uint) (int, error) {
	return gorm.G[Room](rd.DB).Where("id=?", id).Delete(ctx)
}
