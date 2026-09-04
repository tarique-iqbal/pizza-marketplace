package fixtures

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"order-service/internal/domain/readmodel"
	"order-service/tests/testutil"
)

func LoadCustomerFixtures(t *testing.T, db *gorm.DB) []readmodel.Customer {
	phone := "+49 30 12345678"

	customers := []readmodel.Customer{
		{
			ID:        testutil.MustNewID(),
			Email:     "john.doe@example.com",
			FirstName: "John",
			Phone:     &phone,
		},
		{
			ID:        testutil.MustNewID(),
			Email:     "jane.roe@example.com",
			FirstName: "Jane",
		},
	}

	require.NoError(t, db.Create(&customers).Error)

	return customers
}
