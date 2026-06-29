package validator

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
	// register custom validations
	validate.RegisterValidation("date", validateDate)
	validate.RegisterValidation("phone", validatePhone)
}

func Validate(i interface{}) error {
	return validate.Struct(i)
}

func validateDate(fl validator.FieldLevel) bool {
	if fl.Field().String() == "" {
		return true
	}
	matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}$`, fl.Field().String())
	return matched
}

func validatePhone(fl validator.FieldLevel) bool {
	if fl.Field().String() == "" {
		return true
	}
	matched, _ := regexp.MatchString(`^1[3-9]\d{9}$`, fl.Field().String())
	return matched
}

// RegisterCustomValidation allows adding more validators at runtime.
func RegisterCustomValidation(tag string, fn validator.Func) {
	validate.RegisterValidation(tag, fn)
}
