package model

import (
	"strings"

	validation "github.com/go-ozzo/ozzo-validation"
)

func ValidatePassword(password string) error {
	password = strings.TrimSpace(password)
	return validation.Validate(password, validation.Required, validation.Length(6, 100))
}
