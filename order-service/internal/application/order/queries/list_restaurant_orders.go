package queries

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	orderapp "order-service/internal/application/order"
	"order-service/internal/domain/order"
)

type ListRestaurantOrders struct {
	orderRepo order.OrderRepository
}

func NewListRestaurantOrders(orderRepo order.OrderRepository) *ListRestaurantOrders {
	return &ListRestaurantOrders{orderRepo: orderRepo}
}

func (uc *ListRestaurantOrders) Execute(
	ctx context.Context,
	restaurantID, ownerID uuid.UUID,
	cursor string,
	limit int,
) (orderapp.ListOrdersResponse, error) {
	after, err := orderapp.DecodeCursor(cursor)
	if err != nil {
		return orderapp.ListOrdersResponse{}, err
	}

	limit = orderapp.ClampLimit(limit)

	orders, err := uc.orderRepo.ListByRestaurant(ctx, restaurantID, ownerID, after, limit+1)
	if err != nil {
		return orderapp.ListOrdersResponse{}, fmt.Errorf("failed to list orders: %w", err)
	}

	hasMore := len(orders) > limit
	if hasMore {
		orders = orders[:limit]
	}

	items := make([]orderapp.OrderResponse, 0, len(orders))
	for i := range orders {
		items = append(items, orderapp.ToOrderResponse(&orders[i]))
	}

	var nextCursor string
	if hasMore {
		last := orders[len(orders)-1]
		nextCursor = orderapp.EncodeCursor(order.PageCursor{PlacedAt: last.PlacedAt, ID: last.ID})
	}

	return orderapp.ListOrdersResponse{Orders: items, NextCursor: nextCursor}, nil
}
