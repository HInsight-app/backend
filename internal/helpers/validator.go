package helpers

import (
	"github.com/go-playground/validator/v10"
)

// CustomValidator wraps the v10 validator to satisfy Echo's interface requirements
type CustomValidator struct {
	Validator *validator.Validate
}

// Validate executes the validation rules defined in your struct tags
func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.Validator.Struct(i); err != nil {
		return err
	}
	return nil
}

// NewValidator initializes a new instance of the custom validator
func NewValidator() *CustomValidator {
	return &CustomValidator{Validator: validator.New()}
}
