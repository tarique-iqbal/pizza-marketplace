package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/infrastructure/persistence"
	"restaurant-service/tests/infrastructure/db/fixtures"
	"restaurant-service/tests/testutil"
)

type restaurantRepoSetup struct {
	DB             *gorm.DB
	RestaurantRepo restaurant.RestaurantRepository
}

func setupRestaurantRepo(t *testing.T) restaurantRepoSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	_ = fixtures.LoadRestaurantFixtures(t, db.DB)

	restaurantRepo := persistence.NewRestaurantRepository(db.DB)

	return restaurantRepoSetup{
		DB:             db.DB,
		RestaurantRepo: restaurantRepo,
	}
}

func TestRestaurantRepository_Create(t *testing.T) {
	setup := setupRestaurantRepo(t)

	checklist := restaurant.NewChecklist()
	checklist.Complete(restaurant.ChecklistBasic)

	res := restaurant.Restaurant{
		ID:        testutil.MustNewID(),
		OwnerID:   testutil.MustNewID(),
		Name:      "Pizza Paradise",
		VATNumber: "DE323678654",
		Checklist: checklist,
		CreatedAt: time.Now().UTC(),
	}

	err := setup.RestaurantRepo.Create(context.Background(), &res)
	assert.NoError(t, err)
	assert.NotZero(t, res.ID)
	assert.True(t, res.Checklist[restaurant.ChecklistBasic])
}

func TestRestaurantRepository_Update(t *testing.T) {
	setup := setupRestaurantRepo(t)

	var res restaurant.Restaurant
	err := setup.DB.First(&res).Error
	assert.NoError(t, err)

	deliveryKm := int16(5)
	deliveryTimeMin := int16(30)
	deliveryTimeMax := int16(45)

	res.DeliveryKm = &deliveryKm
	res.DeliveryTimeMin = &deliveryTimeMin
	res.DeliveryTimeMax = &deliveryTimeMax
	res.DeliveryType = restaurant.DeliveryOwn
	res.Checklist.Complete(restaurant.ChecklistDelivery)

	err = setup.RestaurantRepo.Update(context.Background(), &res)
	assert.NoError(t, err)

	var r restaurant.Restaurant
	err = setup.DB.Take(&r, "id = ?", res.ID).Error
	assert.NoError(t, err)

	assert.NotNil(t, r.DeliveryKm)
	assert.Equal(t, int16(5), *r.DeliveryKm)
	assert.NotNil(t, r.DeliveryTimeMin)
	assert.Equal(t, int16(30), *r.DeliveryTimeMin)
	assert.NotNil(t, r.DeliveryTimeMax)
	assert.Equal(t, int16(45), *r.DeliveryTimeMax)
	assert.Equal(t, restaurant.DeliveryOwn, r.DeliveryType)
	assert.True(t, r.Checklist[restaurant.ChecklistDelivery])
}

func TestRestaurantRepository_FindBySlug(t *testing.T) {
	setup := setupRestaurantRepo(t)

	res, err := setup.RestaurantRepo.FindBySlug(
		context.Background(),
		"anatolische-kueche", // from fixture
	)
	assert.NoError(t, err)

	assert.NotNil(t, res)
	assert.NotNil(t, res.Slug)
	assert.Equal(t, "anatolische-kueche", *res.Slug)

	res, err = setup.RestaurantRepo.FindBySlug(
		context.Background(),
		"not-exist",
	)
	assert.NoError(t, err)
	assert.Nil(t, res)
}

func TestRestaurantRepository_FindByIDAndOwner(t *testing.T) {
	setup := setupRestaurantRepo(t)

	var existing restaurant.Restaurant
	err := setup.DB.First(&existing).Error
	assert.NoError(t, err)

	tests := []struct {
		name         string
		restaurantID uuid.UUID
		ownerID      uuid.UUID
		found        bool
	}{
		{
			name:         "found",
			restaurantID: existing.ID,
			ownerID:      existing.OwnerID,
			found:        true,
		},
		{
			name:         "wrong owner",
			restaurantID: existing.ID,
			ownerID:      testutil.MustNewID(),
		},
		{
			name:         "wrong restaurant id",
			restaurantID: testutil.MustNewID(),
			ownerID:      existing.OwnerID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := setup.RestaurantRepo.FindByIDAndOwner(
				context.Background(),
				tt.restaurantID,
				tt.ownerID,
			)

			assert.NoError(t, err)

			if !tt.found {
				assert.Nil(t, res)
				return
			}

			assert.NotNil(t, res)
			assert.Equal(t, existing.ID, res.ID)
			assert.Equal(t, existing.OwnerID, res.OwnerID)
		})
	}
}
