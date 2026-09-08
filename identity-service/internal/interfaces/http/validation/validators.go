package validation

import (
	"regexp"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var namePattern = regexp.MustCompile(`^[\p{L} '-]+$`)

func init() {
	if engine, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = engine.RegisterValidation("name", isName)
	}
}

// isName allows Unicode letters, spaces, hyphens, and apostrophes; rejects blank-after-trim.
func isName(fl validator.FieldLevel) bool {
	value := strings.TrimSpace(fl.Field().String())
	if value == "" {
		return false
	}

	return namePattern.MatchString(value)
}
