package order

import (
	"order-service/internal/domain/order"
	"order-service/internal/shared/money"
)

func ToOrderResponse(o *order.Order) OrderResponse {
	items := make([]OrderItemView, 0, len(o.Items))
	for _, item := range o.Items {
		items = append(items, OrderItemView{
			ItemID:          item.ID,
			PizzaID:         item.PizzaID,
			PizzaName:       item.PizzaName,
			SizeID:          item.SizeID,
			DiameterCm:      item.SizeDiameter,
			ExtraToppingIDs: item.ExtraToppingIDs,
			Quantity:        item.Quantity,
			UnitPrice:       money.Money(item.UnitPrice),
			LineTotal:       money.Money(item.TotalPrice),
		})
	}

	return OrderResponse{
		OrderID:         o.ID,
		Status:          o.Status,
		Fulfillment:     o.Fulfillment,
		ContactEmail:    o.ContactEmail,
		ContactPhone:    o.ContactPhone,
		DeliveryAddress: o.DeliveryAddress,
		Items:           items,
		Subtotal:        money.Money(o.Subtotal),
		DeliveryFee:     money.Money(o.DeliveryFee),
		Total:           money.Money(o.Total),
		Currency:        o.Currency,
		PlacedAt:        o.PlacedAt,
		ConfirmedAt:     o.ConfirmedAt,
		ReadyAt:         o.ReadyAt,
		CompletedAt:     o.CompletedAt,
		CancelledAt:     o.CancelledAt,
	}
}
