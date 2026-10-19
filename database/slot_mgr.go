package database

import (
	"context"

	"gorm.io/gorm"
)

type (
	SlotsData struct {
		*gorm.DB
	}

	SlotsMGR interface {
		GetByRoom(ctx context.Context, rid uint) ([]Slot, error)
		GetByID(ctx context.Context, sid uint) (Slot, error)
		AddByRoom(ctx context.Context, slot *Slot) error
		Delete(ctx context.Context, id uint) (int, error)
		UpdateImage(ctx context.Context, slot *Slot) (int, error)
	}
)

// фабрика интерфейса
func (stg Storage) Slots() SlotsMGR {
	var slmgr SlotsMGR = SlotsData{stg.DB}
	return slmgr
}

// отдает слоты, которые принадлежат этой комнате
func (sl SlotsData) GetByRoom(ctx context.Context, rid uint) ([]Slot, error) {
	return gorm.G[Slot](sl.DB).Where("room_id=?", rid).Find(ctx)
}

// добавляет слот
func (sl SlotsData) AddByRoom(ctx context.Context, slot *Slot) error {
	return gorm.G[Slot](sl.DB).Create(ctx, slot)
}

// отдает слоты, которые принадлежат этой комнате
func (sl SlotsData) GetByID(ctx context.Context, sid uint) (Slot, error) {
	return gorm.G[Slot](sl.DB).Where("id=?", sid).First(ctx)
}

// обновляет ссылку на изображение слота
func (sl SlotsData) UpdateImage(ctx context.Context, slot *Slot) (int, error) {
	return gorm.G[Slot](sl.DB).Where("id=?", slot.ID).Update(ctx, "image_path", slot.ImagePath)
}

// удаляет слот из базы
func (sl SlotsData) Delete(ctx context.Context, id uint) (int, error) {
	//тут надо вызвать метод удаляющий изображение с диска
	return gorm.G[Slot](sl.DB).Where("id=?", id).Delete(ctx)
}
