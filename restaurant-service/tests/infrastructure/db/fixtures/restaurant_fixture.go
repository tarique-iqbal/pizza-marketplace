package fixtures

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"restaurant-service/internal/domain/payout"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/tests/testutil"
)

func LoadRestaurantFixtures(t *testing.T, db *gorm.DB) error {
	checklist := restaurant.NewChecklist()
	checklist.Complete(restaurant.ChecklistBasic)

	restaurants := []restaurant.Restaurant{
		{
			Name:      "Pizza Paradise",
			VATNumber: "DE987687654",
			Checklist: checklist,
			CreatedAt: time.Now().UTC(),
		},
		{
			Name:      "Anatolische Küche",
			VATNumber: "DE987321321",
			Slug:      testutil.StringPtr("anatolische-kueche"),
			Email:     testutil.StringPtr("kontakt@anatolisch.de"),
			Phone:     testutil.StringPtr("+49 40 76543210"),
			Website:   testutil.StringPtr("https://anatolisch.de"),
			Checklist: restaurant.Checklist{
				restaurant.ChecklistBasic:        true,
				restaurant.ChecklistContact:      true,
				restaurant.ChecklistAddress:      true,
				restaurant.ChecklistDelivery:     true,
				restaurant.ChecklistPayout:       true,
				restaurant.ChecklistOpeningHours: true,
			},
			Status: restaurant.StatusDraft,
			Address: restaurant.Address{
				House:      "12",
				Street:     "Musterstraße",
				PostalCode: "20095",
				City:       "Hamburg",
			},
			Lat:      testutil.Float64Ptr(53.5511),
			Lon:      testutil.Float64Ptr(9.9937),
			Timezone: testutil.StringPtr("Europe/Berlin"),
			OpeningHours: restaurant.OpeningHours{
				Monday:    []restaurant.DayRange{{Open: "11:00", Close: "22:00"}},
				Tuesday:   []restaurant.DayRange{{Open: "11:00", Close: "22:00"}},
				Wednesday: []restaurant.DayRange{{Open: "11:00", Close: "22:00"}},
				Thursday:  []restaurant.DayRange{{Open: "11:00", Close: "22:00"}},
				Friday:    []restaurant.DayRange{{Open: "11:00", Close: "23:00"}},
				Saturday:  []restaurant.DayRange{{Open: "12:00", Close: "23:00"}},
				Sunday:    []restaurant.DayRange{{Open: "12:00", Close: "21:00"}},
			},
			Tags:         []restaurant.RestaurantTag{restaurant.TagVegetarian, restaurant.TagVegan, restaurant.TagHalal},
			Pickup:       true,
			Currency:     "EUR",
			DeliveryType: restaurant.DeliveryOwn,
			DeliveryKm:   testutil.Int16Ptr(7),
			DeliveryFee:  decimal.NewFromFloat(2.50),
			MinimumOrder: decimal.NewFromFloat(18.00),
			Rating:       4.6,
			TotalReviews: 128,
			CreatedAt:    time.Now().UTC(),
			UpdatedAt:    testutil.TimePtr(time.Now().UTC()),
		},
	}

	for _, r := range restaurants {
		r.ID = testutil.MustNewID()
		r.OwnerID = testutil.MustNewID()

		err := db.Create(&r).Error
		require.NoError(t, err)

		if r.Checklist[restaurant.ChecklistPayout] {
			pd, err := payout.NewPayoutDetails(
				r.ID,
				"Mehmet Yilmaz",
				"DE89370400440532013000",
				"DEUTDEFF",
				"Deutsche Bank",
			)
			require.NoError(t, err)

			pd.Status = payout.PayoutActive

			err = db.Create(pd).Error
			require.NoError(t, err)
		}
	}

	return nil
}
