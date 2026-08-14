package fixtures

import (
	"identity-service/internal/domain/user"
	"identity-service/internal/infrastructure/security"
	"identity-service/tests/testutil"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func LoadUserFixtures(t *testing.T, db *gorm.DB) error {
	hasher := security.NewPasswordHasher()
	password, err := hasher.Hash("plainPassword")
	require.NoError(t, err)

	users := []user.User{
		{
			FirstName: "John",
			LastName:  "Doe",
			Email:     "john.doe@example.com",
			Password:  password,
			Role:      "customer",
		},
		{
			FirstName: "Existing",
			LastName:  "User",
			Email:     "existing@example.com",
			Password:  password,
			Role:      "owner",
		},
	}

	for _, u := range users {
		u.ID = testutil.MustNewID()
		err = db.Create(&u).Error
		require.NoError(t, err)
	}

	return nil
}
