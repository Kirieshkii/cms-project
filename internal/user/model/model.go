// Package model содержит структуры и методы для работы с пользователями в системе.
// Он включает в себя валидацию данных пользователя, управление паролями, а также
// функции для безопасного хэширования и сравнения паролей.
package model

import (
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID                int    `json:"id"`
	Email             string `json:"email"`
	EncryptedPassword string `json:"-"`
}

// Validate валидирует полученного пользователя по email и длине password
func (u *User) ValidateEmail() error {
	return validation.ValidateStruct(u, validation.Field(&u.Email, validation.Required, is.Email))
}

// BeforeCreate шифрует пароль, полученный в структуре User, в случае неудачи возвращает ошибку
func (u *User) BeforeCreate(password string) error {
	if len(password) > 0 {
		enc, err := encryptString(password)
		if err != nil {
			return err
		}

		u.EncryptedPassword = enc
	}

	return nil
}

// ComparePassword сравнивает хэш пароли
func (u *User) ComparePassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.EncryptedPassword), []byte(password)) == nil
}

// encryptString генерирует хэш пароль
func encryptString(s string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(s), bcrypt.MinCost)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
