package validation

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func init() {
	if engine, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = engine.RegisterValidation("iban", isIban)
	}
}

// isIban validates the ISO 13616 structure and mod-97 checksum of an IBAN.
func isIban(fl validator.FieldLevel) bool {
	iban := strings.ToUpper(strings.ReplaceAll(fl.Field().String(), " ", ""))

	if len(iban) < 15 || len(iban) > 34 {
		return false
	}

	for i, r := range iban {
		if i < 2 {
			if r < 'A' || r > 'Z' {
				return false
			}
			continue
		}
		if (r < '0' || r > '9') && (r < 'A' || r > 'Z') {
			return false
		}
	}

	rearranged := iban[4:] + iban[:4]

	var numeric strings.Builder
	for _, r := range rearranged {
		if r >= 'A' && r <= 'Z' {
			numeric.WriteString(strconv.Itoa(int(r-'A') + 10))
			continue
		}
		numeric.WriteRune(r)
	}

	return mod97(numeric.String()) == 1
}

func mod97(numeric string) int {
	remainder := 0
	for _, r := range numeric {
		remainder = (remainder*10 + int(r-'0')) % 97
	}
	return remainder
}
