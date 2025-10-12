package service

import (
	"fmt"

	storage "github.com/Kirieshkii/cms-project/internal/store"
	"github.com/Kirieshkii/cms-project/internal/user/model"
)

func CreateAdmin(s storage.Store, email string, password string) error {

	u := &model.User{
		Email: email,
	}

	if err := u.ValidateEmail(); err != nil {
		return fmt.Errorf("ошибка валидации email: %w", err)
	}

	if err := model.ValidatePassword(password); err != nil {
		return fmt.Errorf("ошибка валидации password: %w", err)
	}

	if err := u.BeforeCreate(password); err != nil {
		return fmt.Errorf("ошибка хеширования пароля: %w", err)
	}

	if err := s.User().Create(u); err != nil {
		return fmt.Errorf("ошибка записи в БД: %w", err)
	}

	return nil
}
