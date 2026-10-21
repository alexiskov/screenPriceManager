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
		Add(ctx context.Context, room *Room) error
		GetAll(ctx context.Context) ([]Room, error)
		GetByID(ctx context.Context, id uint) (Room, error)
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

func (rd RoomsData) GetByID(ctx context.Context, id uint) (Room, error) {
	return gorm.G[Room](rd.DB).Where("id=?", id).First(ctx)
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

// удалит запись о комнате из базы данных + все ее слоты
func (rd RoomsData) Delete(ctx context.Context, id uint) (int, error) {
	var rowsAffected int
	err := rd.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Физическое удаление слотов (оставляем как есть, эта часть корректна)
		if err := tx.Unscoped().Where("room_id = ?", id).Delete(&Slot{}).Error; err != nil {
			return err
		}

		// 2. Ключевое изменение: используем классический API для физического удаления комнаты
		res := tx.Unscoped().Where("id = ?", id).Delete(&Room{})
		if res.Error != nil {
			return res.Error
		}
		rowsAffected = int(res.RowsAffected)
		return nil
	})
	if err != nil {
		return 0, err
	}
	if rowsAffected == 0 {
		return 0, gorm.ErrRecordNotFound
	}
	return rowsAffected, nil
}
