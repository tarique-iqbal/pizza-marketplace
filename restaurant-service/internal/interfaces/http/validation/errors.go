package validation

import (
	"errors"

	"github.com/go-playground/validator/v10"
)

type ErrorMsg struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func getErrorMsg(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "This field is required."
	case "email":
		return "Please provide a valid email address."
	case "min":
		return "Minimum length required: " + fe.Param()
	case "max":
		return "Maximum length allowed: " + fe.Param()
	case "gte":
		return "Must be greater than or equal to: " + fe.Param()
	case "lte":
		return "Must be less than or equal to: " + fe.Param()
	case "oneof":
		return "Must be one of: " + fe.Param()
	}
	return "Unknown error"
}

func ExtractValidationErrors(err error) []ErrorMsg {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		out := make([]ErrorMsg, len(ve))
		for i, fe := range ve {
			out[i] = ErrorMsg{Field: fe.Field(), Message: getErrorMsg(fe)}
		}
		return out
	}
	return nil
}
