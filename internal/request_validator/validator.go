package request_validator

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"boilerplate-api/internal/api_errors"
	"boilerplate-api/internal/constants"

	"github.com/go-playground/validator/v10"
)

// Validator structure
type Validator struct {
	*validator.Validate
}

// NewValidator Register Custom Validators
func NewValidator() Validator {
	v := validator.New()
	_ = v.RegisterValidation("phone", func(fl validator.FieldLevel) bool {
		if fl.Field().String() != "" {
			match, _ := regexp.MatchString("^[- +()]*[0-9][- +()0-9]*$", fl.Field().String())
			return match
		}
		return true
	})
	_ = v.RegisterValidation("gender", func(fl validator.FieldLevel) bool {
		if fl.Field().String() != "" {
			var valType constants.Gender
			if err := valType.IsValidVal(fl.Field().String()); err != nil {
				return false
			}
		}
		return true
	})
	_ = v.RegisterValidation("email", func(fl validator.FieldLevel) bool {
		if fl.Field().String() != "" {
			match, _ := regexp.MatchString(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`, fl.Field().String())
			return match
		}
		return true
	})
	_ = v.RegisterValidation("date", func(fl validator.FieldLevel) bool {
		_, parseErr := time.Parse("2006-01-02", fl.Field().String())
		return parseErr == nil
	})
	v.RegisterAlias("optional", "omitempty")

	return Validator{
		Validate: v,
	}
}

func (cv Validator) generateValidationMessage(field string, rule string, param string) (message string) {
	switch rule {
	case "required":
		return fmt.Sprintf("%s is %s", field, rule)
	case "phone":
		return fmt.Sprintf("Field %s is not valid", field)
	case "gender":
		return fmt.Sprintf("Field '%s' is not valid", field)
	case "email":
		return fmt.Sprintf("Field '%s' is not valid", field)
	case "date":
		return fmt.Sprintf("Invalid date format for '%s' use YYYY-MM-DD", field)
	case "required_if":
		params := strings.Split(param, " ")
		requiredIfParent := params[0]
		values := ""
		paramsLen := len(params)
		if paramsLen > 1 {
			values = strings.Join(params[1:paramsLen], ",")
		}
		return fmt.Sprintf("Field %s is required when %s is %s", field, requiredIfParent, values)
	default:
		return fmt.Sprintf("Field '%s' is not valid.", field)
	}
}

func (cv Validator) GenerateValidationResponse(err error) []api_errors.ValidationError {
	var validations []api_errors.ValidationError
	for _, value := range err.(validator.ValidationErrors) {
		field, rule := value.Field(), value.Tag()
		validation := api_errors.ValidationError{Field: field, Message: cv.generateValidationMessage(field, rule, value.Param())}
		validations = append(validations, validation)
	}
	return validations
}
