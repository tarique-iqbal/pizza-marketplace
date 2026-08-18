package pizza_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	pizzaapp "restaurant-service/internal/application/pizza"
	"restaurant-service/internal/shared/money"
)

func TestPizzaResponse_HasActivePrice(t *testing.T) {
	tests := []struct {
		name   string
		prices []pizzaapp.PizzaPriceResponse
		want   bool
	}{
		{name: "no prices", prices: nil, want: false},
		{
			name: "one active price",
			prices: []pizzaapp.PizzaPriceResponse{
				{SizeID: uuid.New(), Price: money.Money(decimal.NewFromInt(10)), IsActive: true},
			},
			want: true,
		},
		{
			name: "all inactive",
			prices: []pizzaapp.PizzaPriceResponse{
				{SizeID: uuid.New(), Price: money.Money(decimal.NewFromInt(10)), IsActive: false},
				{SizeID: uuid.New(), Price: money.Money(decimal.NewFromInt(12)), IsActive: false},
			},
			want: false,
		},
		{
			name: "mixed active and inactive",
			prices: []pizzaapp.PizzaPriceResponse{
				{SizeID: uuid.New(), Price: money.Money(decimal.NewFromInt(10)), IsActive: false},
				{SizeID: uuid.New(), Price: money.Money(decimal.NewFromInt(12)), IsActive: true},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := pizzaapp.PizzaResponse{Prices: tt.prices}
			assert.Equal(t, tt.want, p.HasActivePrice())
		})
	}
}
