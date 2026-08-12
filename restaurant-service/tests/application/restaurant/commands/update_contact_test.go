package commands_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/application/restaurant/commands"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/infrastructure/persistence"
	apperr "restaurant-service/internal/shared/errors"
	"restaurant-service/tests/infrastructure/db/fixtures"
	"restaurant-service/tests/testutil"
)

type updateContactSetup struct {
	DB            *gorm.DB
	UpdateContact *commands.UpdateContact
}

func setupUpdateContact(t *testing.T) updateContactSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	_ = fixtures.LoadRestaurantFixtures(t, db.DB)

	restaurantRepo := persistence.NewRestaurantRepository(db.DB)
	payoutDetailsRepo := persistence.NewPayoutDetailsRepository(db.DB)
	updateContact := commands.NewUpdateContact(restaurantRepo, payoutDetailsRepo, testutil.NoopPublisher{})

	return updateContactSetup{
		DB:            db.DB,
		UpdateContact: updateContact,
	}
}

func validContactInput() resapp.UpdateContactRequest {
	return resapp.UpdateContactRequest{
		Email:   testutil.StringPtr("owner@example.com"),
		Phone:   testutil.StringPtr("+49 40 12345678"),
		Website: testutil.StringPtr("https://example.com"),
	}
}

func TestUpdateContact_Success(t *testing.T) {
	env := setupUpdateContact(t)

	res := firstRestaurant(t, env.DB)

	assert.False(t, res.Checklist[restaurant.ChecklistContact])

	output, err := env.UpdateContact.Execute(
		context.Background(),
		res.ID,
		res.OwnerID,
		validContactInput(),
	)

	require.NoError(t, err)

	assert.Equal(t, "owner@example.com", *output.Contact.Email)
	assert.Equal(t, "+49 40 12345678", *output.Contact.Phone)
	assert.Equal(t, "https://example.com", *output.Contact.Website)

	var updated restaurant.Restaurant

	err = env.DB.Take(&updated, "id = ?", res.ID).Error
	require.NoError(t, err)

	assert.True(t, updated.Checklist[restaurant.ChecklistContact])
	assert.Equal(t, "owner@example.com", *updated.Email)
	assert.Equal(t, "+49 40 12345678", *updated.Phone)
	assert.Equal(t, "https://example.com", *updated.Website)
	assert.False(t, updated.UpdatedAt.IsZero())
}

func TestUpdateContact_RestaurantNotOwned(t *testing.T) {
	env := setupUpdateContact(t)

	res := firstRestaurant(t, env.DB)

	otherOwnerID := uuid.New()

	_, err := env.UpdateContact.Execute(
		context.Background(),
		res.ID,
		otherOwnerID,
		validContactInput(),
	)

	require.Error(t, err)

	assert.Contains(t, err.Error(), "access denied")
	assert.ErrorIs(t, err, apperr.ErrForbidden)
}

func TestUpdateContact_RestaurantNotFound(t *testing.T) {
	env := setupUpdateContact(t)

	_, err := env.UpdateContact.Execute(
		context.Background(),
		uuid.New(),
		uuid.New(),
		validContactInput(),
	)

	require.Error(t, err)

	assert.Contains(t, err.Error(), "access denied")
	assert.ErrorIs(t, err, apperr.ErrForbidden)
}

func TestUpdateContact_WebsiteOmitted_ClearsWebsite(t *testing.T) {
	env := setupUpdateContact(t)

	res := firstRestaurant(t, env.DB)

	input := resapp.UpdateContactRequest{
		Email:   testutil.StringPtr("owner@example.com"),
		Phone:   testutil.StringPtr("+49 40 12345678"),
		Website: nil,
	}

	output, err := env.UpdateContact.Execute(
		context.Background(),
		res.ID,
		res.OwnerID,
		input,
	)

	require.NoError(t, err)

	assert.Equal(t, "owner@example.com", *output.Contact.Email)
	assert.Equal(t, "+49 40 12345678", *output.Contact.Phone)
	assert.Nil(t, output.Contact.Website)

	var updated restaurant.Restaurant

	err = env.DB.Take(&updated, "id = ?", res.ID).Error
	require.NoError(t, err)

	assert.Nil(t, updated.Website)
}
