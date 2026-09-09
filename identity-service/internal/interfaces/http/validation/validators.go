package validation

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var namePattern = regexp.MustCompile(`^[\p{L} '-]+$`)

const asciiSymbols = "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"

func init() {
	if engine, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = engine.RegisterValidation("name", isName)
		_ = engine.RegisterValidation("password", isPassword)
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

// isPassword requires a lowercase letter, an uppercase letter, a digit, and an ASCII symbol.
func isPassword(fl validator.FieldLevel) bool {
	var hasLower, hasUpper, hasDigit, hasSymbol bool

	for _, r := range fl.Field().String() {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		case strings.ContainsRune(asciiSymbols, r):
			hasSymbol = true
		}
	}

	return hasLower && hasUpper && hasDigit && hasSymbol
}
