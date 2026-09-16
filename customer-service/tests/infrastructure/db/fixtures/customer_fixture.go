package fixtures

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"customer-service/internal/domain/customer"
	"customer-service/tests/testutil"
)

func LoadCustomerFixtures(t *testing.T, db *gorm.DB) []customer.Customer {
	phone := "+49 40 12345678"

	customers := []customer.Customer{
		{
			ID:        testutil.MustNewID(),
			Email:     "no-phone@example.com",
			FirstName: "Nina",
			LastName:  "Nolan",
		},
		{
			ID:        testutil.MustNewID(),
			Email:     "with-phone@example.com",
			FirstName: "Peter",
			LastName:  "Park",
			Phone:     &phone,
		},
	}

	for i := range customers {
		require.NoError(t, db.Create(&customers[i]).Error)
	}

	return customers
}
