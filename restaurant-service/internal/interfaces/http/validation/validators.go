package validation

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"restaurant-service/internal/application/restaurant"
)

var hhmmPattern = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)
var phonePattern = regexp.MustCompile(`^\+?[0-9()\-\s]{6,32}$`)

func init() {
	if engine, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = engine.RegisterValidation("iban", isIban)
		_ = engine.RegisterValidation("hhmm", isHHMM)
		_ = engine.RegisterValidation("phone", isPhone)
		engine.RegisterStructValidation(validateDayRange, restaurant.DayRangeRequest{})
		engine.RegisterStructValidation(validateDelivery, restaurant.UpdateDeliveryRequest{})
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

func isHHMM(fl validator.FieldLevel) bool {
	return hhmmPattern.MatchString(fl.Field().String())
}

// isPhone accepts digits with optional leading +, spaces, hyphens, and
// parentheses, requiring at least 6 digits so strings like "asdf" or
// all-separator input don't pass.
func isPhone(fl validator.FieldLevel) bool {
	value := fl.Field().String()

	if !phonePattern.MatchString(value) {
		return false
	}

	digits := 0
	for _, r := range value {
		if r >= '0' && r <= '9' {
			digits++
		}
	}

	return digits >= 6
}

func validateDayRange(sl validator.StructLevel) {
	dr := sl.Current().Interface().(restaurant.DayRangeRequest)

	if !hhmmPattern.MatchString(dr.Open) || !hhmmPattern.MatchString(dr.Close) {
		return
	}

	if dr.Open >= dr.Close {
		sl.ReportError(reflect.ValueOf(dr.Close), "Close", "Close", "gtfield_open", "")
	}
}

func validateDelivery(sl validator.StructLevel) {
	req := sl.Current().Interface().(restaurant.UpdateDeliveryRequest)

	if req.DeliveryType != "none" {
		if req.DeliveryKm == nil {
			sl.ReportError(reflect.ValueOf(req.DeliveryKm), "DeliveryKm", "DeliveryKm", "required_if_delivery", "")
		}
		if req.DeliveryTimeMin == nil {
			sl.ReportError(reflect.ValueOf(req.DeliveryTimeMin), "DeliveryTimeMin", "DeliveryTimeMin", "required_if_delivery", "")
		}
		if req.DeliveryTimeMax == nil {
			sl.ReportError(reflect.ValueOf(req.DeliveryTimeMax), "DeliveryTimeMax", "DeliveryTimeMax", "required_if_delivery", "")
		}
	}

	if req.DeliveryTimeMin != nil && req.DeliveryTimeMax != nil && *req.DeliveryTimeMax <= *req.DeliveryTimeMin {
		sl.ReportError(reflect.ValueOf(req.DeliveryTimeMax), "DeliveryTimeMax", "DeliveryTimeMax", "gtfield_deliverytimemax", "")
	}
}
