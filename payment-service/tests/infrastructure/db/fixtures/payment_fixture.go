package fixtures

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"payment-service/internal/domain/payment"
	"payment-service/tests/testutil"
)

func LoadPaymentFixtures(t *testing.T, db *gorm.DB) []payment.Payment {
	succeededGatewayPaymentID := "tr_test_succeeded"

	payments := []payment.Payment{
		{
			ID:           testutil.MustNewID(),
			SubjectType:  "order",
			SubjectID:    testutil.MustNewID(),
			RestaurantID: testutil.MustNewID(),
			CustomerID:   testutil.MustNewID(),
			Amount:       decimal.NewFromFloat(24.50),
			Currency:     "EUR",
			PlatformFee:  decimal.NewFromFloat(1.50),
			Status:       payment.StatusPending,
			Gateway:      "mollie",
		},
		{
			ID:               testutil.MustNewID(),
			SubjectType:      "order",
			SubjectID:        testutil.MustNewID(),
			RestaurantID:     testutil.MustNewID(),
			CustomerID:       testutil.MustNewID(),
			Amount:           decimal.NewFromFloat(12.00),
			Currency:         "EUR",
			PlatformFee:      decimal.NewFromFloat(0.80),
			Status:           payment.StatusSucceeded,
			Gateway:          "mollie",
			GatewayPaymentID: &succeededGatewayPaymentID,
		},
	}

	for i := range payments {
		require.NoError(t, db.Create(&payments[i]).Error)
	}

	return payments
}
