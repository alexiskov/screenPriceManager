package database

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

type (
	UserData struct {
		*gorm.DB
	}

	UserMGR interface {
		SetAsRoot(ctx context.Context, userEmail, userPassword string) error
		GetByCredentials(ctx context.Context, useremil, userpassword string) (User, error)
		UpdateUser(ctx context.Context, user User) error
	}
)

func (stg Storage) Users() UserMGR {
	var umgr UserMGR = &UserData{stg.DB}
	return umgr
}

// задает пользователю root учетные данные полученные из параметров
func (um *UserData) SetAsRoot(ctx context.Context, userEmail, userPassword string) error {
	_, err := gorm.G[User](um.DB).Where("id=1").First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err := gorm.G[User](um.DB).Create(ctx, &User{Email: userEmail, Password: userPassword, Name: "root"})
			if err != nil {
				return fmt.Errorf("root user creating error: %w", err)
			}
		} else {
			return fmt.Errorf("root user finding error: %w", err)
		}
	}
	_, err = gorm.G[map[string]interface{}](um.DB).Table("users").Where("id=1").Updates(ctx, map[string]interface{}{"email": userEmail, "password": userPassword})
	if err != nil {
		return fmt.Errorf("rouut user credentials from config data updating error: %w", err)
	}
	return nil
}

func (um *UserData) GetByCredentials(ctx context.Context, useremail, userpassword string) (User, error) {
	return gorm.G[User](um.DB).Where("email=? AND password=?", useremail, userpassword).First(ctx)
}

func (um *UserData) UpdateUser(ctx context.Context, user User) error {
	_, err := gorm.G[User](um.DB).Where("id=?", user.ID).Update(ctx, "auth_hash", user.AuthHash)
	if err != nil {
		return fmt.Errorf("user auth ahsh updating error: %w", err)
	}
	return nil
}
