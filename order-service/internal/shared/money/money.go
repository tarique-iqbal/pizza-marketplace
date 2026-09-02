package money

import "github.com/shopspring/decimal"

// Money wraps decimal.Decimal to preserve trailing zeros in JSON price values.
type Money decimal.Decimal

func (m Money) MarshalJSON() ([]byte, error) {
	return []byte(`"` + decimal.Decimal(m).StringFixed(2) + `"`), nil
}

func (m *Money) UnmarshalJSON(data []byte) error {
	var d decimal.Decimal
	if err := d.UnmarshalJSON(data); err != nil {
		return err
	}

	*m = Money(d)

	return nil
}
