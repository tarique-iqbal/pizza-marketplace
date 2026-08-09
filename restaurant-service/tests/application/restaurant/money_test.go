package restaurant_test

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	resapp "restaurant-service/internal/application/restaurant"
)

func TestMoney_MarshalJSON_KeepsTrailingZeros(t *testing.T) {
	cases := map[string]string{
		"1.50": `"1.50"`,
		"1.00": `"1.00"`,
		"1.5":  `"1.50"`,
		"1":    `"1.00"`,
		"0":    `"0.00"`,
		"9.99": `"9.99"`,
	}

	for input, want := range cases {
		m := resapp.Money(decimal.RequireFromString(input))

		out, err := json.Marshal(m)
		require.NoError(t, err)
		assert.Equal(t, want, string(out), "input %q", input)
	}
}

func TestMoney_UnmarshalJSON_RoundTrips(t *testing.T) {
	var m resapp.Money
	require.NoError(t, json.Unmarshal([]byte(`"1.50"`), &m))

	out, err := json.Marshal(m)
	require.NoError(t, err)
	assert.Equal(t, `"1.50"`, string(out))
}
