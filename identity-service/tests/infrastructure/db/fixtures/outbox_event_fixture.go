package fixtures

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"identity-service/internal/domain/outbox"
	"identity-service/tests/testutil"
)

func LoadOutboxEventFixtures(t *testing.T, db *gorm.DB) error {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for range 5 {
		restaurantID, payload, err := restaurantPayload(r)
		require.NoError(t, err)

		event, err := outbox.NewOutboxEvent(
			restaurantID,
			outbox.EventRestaurantInitiated,
			payload,
		)
		require.NoError(t, err)

		err = db.Create(&event).Error
		require.NoError(t, err)
	}

	return nil
}

func restaurantPayload(r *rand.Rand) (restaurantID uuid.UUID, payload []byte, err error) {
	restaurantID = testutil.MustNewID()

	payloadMap := map[string]any{
		"restaurant_id": restaurantID,
		"owner_id":      testutil.MustNewID(),
		"business_name": randomString(r, "Restaurant"),
		"vat_number":    randomVAT(r),
	}

	payload, err = json.Marshal(payloadMap)
	if err != nil {
		return restaurantID, nil, err
	}

	return restaurantID, payload, nil
}

func randomString(r *rand.Rand, prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, r.Intn(1_000_000))
}

func randomVAT(r *rand.Rand) string {
	return fmt.Sprintf("DE%09d", r.Intn(1_000_000_000))
}
