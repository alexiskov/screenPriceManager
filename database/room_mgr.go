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
		Rename(ctx context.Context, room *Room) error
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
	result := gorm.WithResult()
	return gorm.G[Room](rd.DB, result).Create(ctx, room)
}

// переименует комнату в базе
func (rd RoomsData) Rename(ctx context.Context, room *Room) error {
	roomRes, err := gorm.G[Room](rd.DB).Raw("UPDATE rooms SET title = ? WHERE id = ? RETURNING *", room.Title, room.ID).First(ctx)
	if err != nil {
		return err
	}
	*room = roomRes
	return nil
}

// удалит запись о комнате из базы данных
func (rd RoomsData) Delete(ctx context.Context, id uint) (int, error) {
	return gorm.G[Room](rd.DB).Where("id=?", id).Delete(ctx)
}
