package user_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	userapp "identity-service/internal/application/user"
	"identity-service/internal/domain/user"
	"identity-service/internal/infrastructure/persistence"
	apperr "identity-service/internal/shared/errors"
	"identity-service/tests/infrastructure/db/fixtures"
	"identity-service/tests/testutil"
)

func setupFindByID(t *testing.T) *userapp.FindByID {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableUser)

	_ = fixtures.LoadUserFixtures(t, db.DB)

	userRepo := persistence.NewUserRepository(db.DB)
	mockPublisher = &MockEventPublisher{}

	return userapp.NewFindByID(userRepo)
}

func TestFindByID_Success(t *testing.T) {
	ctx := context.Background()
	db := testutil.DB(t)
	uc := setupFindByID(t)

	u := &user.User{
		FirstName: "Tony",
		LastName:  "Soprano",
		Email:     "tony@satrialis.com",
		Role:      "owner",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	}

	userID := testutil.MustNewID()
	u.ID = userID

	err := db.DB.WithContext(ctx).Create(u).Error
	require.NoError(t, err)

	res, err := uc.Execute(ctx, userID)

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, userID, res.ID)
	assert.Equal(t, "Tony", res.Name.First)
	assert.Equal(t, "Soprano", res.Name.Last)
	assert.Equal(t, "tony@satrialis.com", res.Email)
	assert.Equal(t, "owner", res.Role)
	assert.Equal(t, "active", res.Status)
}

func TestFindByID_NotFound(t *testing.T) {
	ctx := context.Background()
	uc := setupFindByID(t)

	userID := testutil.MustNewID()

	res, err := uc.Execute(ctx, userID)

	require.Error(t, err)
	assert.True(t, errors.Is(err, apperr.ErrNotFound))
	assert.Equal(t, userapp.Response{}, res)
}
