package handlers_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	orderapp "order-service/internal/application/order"
	"order-service/internal/application/order/handlers"
	"order-service/internal/domain/order"
	"order-service/internal/domain/outbox"
	"order-service/internal/domain/readmodel"
	"order-service/internal/infrastructure/persistence"
	"order-service/tests/testutil"
)

func seedPendingOrder(t *testing.T) (*testutil.TestDB, readmodel.Restaurant, *order.Order) {
	t.Helper()

	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant, testutil.TableOutboxEvent)

	restaurant := readmodel.Restaurant{
		ID:           testutil.MustNewID(),
		OwnerID:      testutil.MustNewID(),
		Name:         "Pizza Paradise",
		OwnerEmail:   "owner@pizzaparadise.de",
		Lat:          53.5511,
		Lon:          9.9937,
		DeliveryFee:  decimal.Zero,
		MinimumOrder: decimal.Zero,
		Pickup:       true,
		DeliveryType: readmodel.DeliveryNone,
		Currency:     "EUR",
		UpdatedAt:    time.Now().UTC(),
	}
	require.NoError(t, db.DB.Create(&restaurant).Error)

	ord := order.NewOrder(
		testutil.MustNewID(),
		testutil.MustNewID(),
		restaurant.ID,
		order.FulfillmentPickup,
		"customer@example.com",
		nil,
		nil,
		nil,
		nil,
		[]order.OrderItem{{
			ID:           testutil.MustNewID(),
			PizzaID:      testutil.MustNewID(),
			SizeID:       testutil.MustNewID(),
			PizzaName:    "Margherita",
			SizeDiameter: 26,
			Quantity:     2,
			UnitPrice:    decimal.NewFromFloat(7.50),
			TotalPrice:   decimal.NewFromFloat(15.00),
		}},
		decimal.NewFromFloat(15.00),
		decimal.Zero,
		decimal.NewFromFloat(15.00),
		"EUR",
	)
	require.NoError(t, persistence.NewOrderRepository(db.DB).Create(context.Background(), ord))

	return db, restaurant, ord
}

func newPaymentEventPayload(t *testing.T, eventName, subjectType string, orderID uuid.UUID) readmodel.EventPayload {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"payment_id":    testutil.MustNewID(),
		"subject_type":  subjectType,
		"subject_id":    orderID,
		"restaurant_id": testutil.MustNewID(),
		"amount":        "15.00",
		"currency":      "EUR",
		"occurred_at":   time.Now().UTC(),
	})
	require.NoError(t, err)

	return readmodel.EventPayload{Name: eventName, Data: body}
}

func firstOutboxEvent(t *testing.T, db *gorm.DB, aggregateID uuid.UUID, eventName string) outbox.OutboxEvent {
	t.Helper()

	var found outbox.OutboxEvent
	err := db.Where("aggregate_id = ? AND event_name = ?", aggregateID, eventName).First(&found).Error
	require.NoError(t, err)

	return found
}

func newPaymentSucceededHandler(db *gorm.DB) *handlers.PaymentSucceededHandler {
	return handlers.NewPaymentSucceededHandler(
		db,
		persistence.NewOrderRepository(db),
		persistence.NewOutboxRepository(db),
		persistence.NewRestaurantRepository(db),
	)
}

func TestPaymentSucceededHandler_Handle_ConfirmsOrderAndDispatchesEvent(t *testing.T) {
	db, restaurant, ord := seedPendingOrder(t)
	h := newPaymentSucceededHandler(db.DB)

	err := h.Handle(newPaymentEventPayload(t, "payment.succeeded", "order", ord.ID))
	require.NoError(t, err)

	updated, err := persistence.NewOrderRepository(db.DB).FindByID(context.Background(), ord.ID)
	require.NoError(t, err)
	assert.Equal(t, order.StatusConfirmed, updated.Status)

	stored := firstOutboxEvent(t, db.DB, ord.ID, "order.confirmed")

	var payload orderapp.OrderConfirmedPayload
	require.NoError(t, json.Unmarshal(stored.Payload, &payload))

	assert.Equal(t, ord.ID, payload.OrderID)
	assert.Equal(t, restaurant.Name, payload.RestaurantName)
	assert.Equal(t, restaurant.OwnerEmail, payload.OwnerEmail)
	assert.Equal(t, "customer@example.com", payload.CustomerEmail)
	assert.Equal(t, "15.00", payload.Total)
	require.Len(t, payload.Items, 1)
	assert.Equal(t, "Margherita", payload.Items[0].Name)
	assert.Equal(t, int16(2), payload.Items[0].Quantity)
}

func TestPaymentSucceededHandler_Handle_NoOpWhenAlreadyConfirmed(t *testing.T) {
	db, _, ord := seedPendingOrder(t)
	h := newPaymentSucceededHandler(db.DB)

	require.NoError(t, h.Handle(newPaymentEventPayload(t, "payment.succeeded", "order", ord.ID)))
	require.NoError(t, h.Handle(newPaymentEventPayload(t, "payment.succeeded", "order", ord.ID)))

	var events []outbox.OutboxEvent
	require.NoError(t, db.DB.Where("aggregate_id = ?", ord.ID).Find(&events).Error)
	assert.Len(t, events, 1, "redelivery must not create a second outbox row")
}

func TestPaymentSucceededHandler_Handle_IgnoresNonOrderSubject(t *testing.T) {
	db, _, ord := seedPendingOrder(t)
	h := newPaymentSucceededHandler(db.DB)

	require.NoError(t, h.Handle(newPaymentEventPayload(t, "payment.succeeded", "subscription", ord.ID)))

	updated, err := persistence.NewOrderRepository(db.DB).FindByID(context.Background(), ord.ID)
	require.NoError(t, err)
	assert.Equal(t, order.StatusPending, updated.Status)
}
