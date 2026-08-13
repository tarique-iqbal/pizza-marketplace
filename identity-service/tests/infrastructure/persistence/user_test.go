package persistence_test

import (
	"context"
	"identity-service/internal/domain/user"
	"identity-service/internal/infrastructure/persistence"
	"identity-service/tests/infrastructure/db/fixtures"
	"identity-service/tests/testutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUserRepo(t *testing.T) user.UserRepository {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableUser)

	_ = fixtures.LoadUserFixtures(t, db.DB)

	return persistence.NewUserRepository(db.DB)
}

func TestUserRepository_Create(t *testing.T) {
	userRepo := setupUserRepo(t)

	usr := user.User{
		FirstName: "Adam",
		LastName:  "D'Angelo",
		Email:     "adam.dangelo@example.com",
		Password:  "hashedpassword",
		Role:      "customer",
		CreatedAt: time.Now().UTC(),
	}

	usr.ID = testutil.MustNewID()

	err := userRepo.Create(context.Background(), &usr)

	assert.Nil(t, err)
	assert.NotZero(t, usr.ID)
}

func TestUserRepository_FindByEmail(t *testing.T) {
	userRepo := setupUserRepo(t)

	usr, err := userRepo.FindByEmail(context.Background(), "john.doe@example.com")
	assert.NoError(t, err)
	assert.Equal(t, "John", usr.FirstName)
}

func TestUserRepository_EmailExists(t *testing.T) {
	userRepo := setupUserRepo(t)

	exists, err := userRepo.EmailExists(context.Background(), "john.doe@example.com")
	assert.NoError(t, err)
	assert.True(t, exists, "Email is expected to be exists")

	exists, err = userRepo.EmailExists(context.Background(), "random@example.com")
	assert.NoError(t, err)
	assert.False(t, exists, "Email is not expected to be exists")
}

func TestUserRepository_FindByID_Success(t *testing.T) {
	ctx := context.Background()
	userRepo := setupUserRepo(t)

	u := &user.User{
		FirstName: "Tony",
		LastName:  "Soprano",
		Email:     "tony@satrialis.com",
		Role:      "owner",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	}

	u.ID = testutil.MustNewID()

	err := userRepo.Create(ctx, u)
	require.NoError(t, err)

	res, err := userRepo.FindByID(ctx, u.ID)

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, u.ID, res.ID)
	assert.Equal(t, u.Email, res.Email)
	assert.Equal(t, u.FirstName, res.FirstName)
	assert.Equal(t, u.LastName, res.LastName)
	assert.Equal(t, u.Role, res.Role)
	assert.Equal(t, u.Status, res.Status)
}

func TestUserRepository_FindByID_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := setupUserRepo(t)

	userID := testutil.MustNewID()

	res, err := repo.FindByID(ctx, userID)

	require.NoError(t, err)
	assert.Nil(t, res)
}
