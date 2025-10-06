package model

import validation "github.com/go-ozzo/ozzo-validation"

func requiredIf(cond bool) validation.RuleFunc {
	return func(value any) error {
		if cond {
			return validation.Validate(value, validation.Required)
		}

		return nil
	}
}
