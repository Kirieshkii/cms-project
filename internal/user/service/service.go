package service

import (
	"context"
	"fmt"
	"log/slog"

	storage "github.com/Kirieshkii/cms-project/internal/store"
	"github.com/Kirieshkii/cms-project/internal/user/model"
)

func CreateAdmin(ctx context.Context, s storage.Store, email string, password string, logger *slog.Logger) error {

	u := &model.User{
		Email: email,
		Role:  "admin", //мб сделать CreateAdmin более универсальной функцией и создавать любого пользователя
	}

	if err := u.ValidateEmail(); err != nil {
		if logger != nil {
			logger.Error("ошибка валидации email",
				"email", email,
				"error", err,
			)
		}
		return fmt.Errorf("ошибка валидации email: %w", err)
	}

	if err := model.ValidatePassword(password); err != nil {
		if logger != nil {
			logger.Error("ошибка валидации password",
				"email", email,
				"error", err,
			)
		}
		return fmt.Errorf("ошибка валидации password: %w", err)
	}

	if err := u.BeforeCreate(password); err != nil {
		if logger != nil {
			logger.Error("ошибка хеширования пароля",
				"email", email,
				"error", err,
			)
		}
		return fmt.Errorf("ошибка хеширования пароля: %w", err)
	}

	if err := s.User().Create(ctx, u); err != nil {
		if logger != nil {
			logger.Error("ошибка создания пользователя в БД",
				"email", email,
				"role", u.Role,
				"error", err,
			)
		}
		return fmt.Errorf("ошибка записи в БД: %w", err)
	}

	// Логируем успешное создание админа (дополнительно к логированию в CLI)
	if logger != nil {
		logger.Info("админ успешно создан через сервис",
			"user_id", u.ID,
			"email", email,
			"role", u.Role,
		)
	}

	return nil
}
