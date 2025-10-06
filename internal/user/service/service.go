package service

import (
	"fmt"

	storage "github.com/Kirieshkii/cms-project/internal/store"
	"github.com/Kirieshkii/cms-project/internal/user/model"
)

func CreateAdmin(s storage.Store, email string, password string) error {

	u := &model.User{
		Email:    email,
		Password: password,
	}

	if err := u.Validate(); err != nil {
		return fmt.Errorf("ошибка валидации email и/или password: %w", err)
	}

	if err := u.BeforeCreate(); err != nil {
		return fmt.Errorf("ошибка хеширования пароля: %w", err)
	}

	if err := s.User().Create(u); err != nil {
		return fmt.Errorf("ошибка создания админа: %w", err)
	}

	return nil
}
