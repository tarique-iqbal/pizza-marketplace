package fixtures

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"customer-service/internal/domain/customer"
	"customer-service/tests/testutil"
)

func LoadAddressFixtures(t *testing.T, db *gorm.DB, customerID uuid.UUID) []customer.Address {
	addresses := []customer.Address{
		{
			ID:         testutil.MustNewID(),
			CustomerID: customerID,
			House:      "10",
			Street:     "Main St",
			City:       "Berlin",
			PostalCode: "10115",
			IsDefault:  true,
		},
		{
			ID:         testutil.MustNewID(),
			CustomerID: customerID,
			House:      "20",
			Street:     "Second St",
			City:       "Berlin",
			PostalCode: "10117",
			IsDefault:  false,
		},
	}

	for i := range addresses {
		require.NoError(t, db.Create(&addresses[i]).Error)
	}

	return addresses
}
