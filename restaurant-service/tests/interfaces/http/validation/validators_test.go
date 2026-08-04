package validation_test

import (
	"testing"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "restaurant-service/internal/interfaces/http/validation"
)

func TestIbanValidator(t *testing.T) {
	engine, ok := binding.Validator.Engine().(*validator.Validate)
	require.True(t, ok)

	cases := []struct {
		name  string
		iban  string
		valid bool
	}{
		{"valid DE", "DE89370400440532013000", true},
		{"valid DE with spaces", "DE89 3704 0044 0532 0130 00", true},
		{"valid GB", "GB29NWBK60161331926819", true},
		{"valid lowercase", "de89370400440532013000", true},
		{"tampered checksum", "DE89370400440532013001", false},
		{"too short", "DE1234", false},
		{"contains hyphens", "DE89-3704-0044-0532-0130-00", false},
		{"empty", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := engine.Var(tc.iban, "iban")
			if tc.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
