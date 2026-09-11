package notification

import (
	"encoding/json"
	"fmt"
	"notification-service/internal/domain/notification"
	"os"
	"strings"
	"time"
	_ "time/tzdata"
)

type OrderConfirmed struct {
	sender   notification.Sender
	template notification.TemplateLoader
}

func NewOrderConfirmed(
	sender notification.Sender,
	template notification.TemplateLoader,
) *OrderConfirmed {
	return &OrderConfirmed{sender: sender, template: template}
}

func (h *OrderConfirmed) Handle(event notification.EventPayload) error {
	var payload struct {
		OrderID        string `json:"order_id"`
		RestaurantID   string `json:"restaurant_id"`
		RestaurantName string `json:"restaurant_name"`
		CustomerEmail  string `json:"customer_email"`
		OwnerEmail     string `json:"owner_email"`
		Items          []struct {
			Name     string `json:"name"`
			Quantity int    `json:"quantity"`
		} `json:"items"`
		Total       string    `json:"total"`
		Currency    string    `json:"currency"`
		ConfirmedAt time.Time `json:"confirmed_at"`
	}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return err
	}

	appName := os.Getenv("APP_NAME")

	confirmedAt := payload.ConfirmedAt
	if loc, err := time.LoadLocation("Europe/Berlin"); err == nil {
		confirmedAt = confirmedAt.In(loc)
	}

	itemLines := make([]string, 0, len(payload.Items))
	for _, item := range payload.Items {
		itemLines = append(itemLines, fmt.Sprintf("%dx %s", item.Quantity, item.Name))
	}

	data := map[string]string{
		"restaurant_name": payload.RestaurantName,
		"order_id":        payload.OrderID,
		"items":           strings.Join(itemLines, ", "),
		"total":           payload.Total,
		"currency":        payload.Currency,
		"confirmed_at":    confirmedAt.Format("Jan 2, 2006 3:04 PM MST"),
		"app_name":        appName,
	}

	if err := h.sendEmail(payload.CustomerEmail, "order_confirmed_customer", data); err != nil {
		return err
	}

	return h.sendEmail(payload.OwnerEmail, "order_confirmed_owner", data)
}

func (h *OrderConfirmed) sendEmail(to, templatePrefix string, data map[string]string) error {
	subject, err := h.template.Render(templatePrefix+"_subject.html", data)
	if err != nil {
		return err
	}

	body, err := h.template.Render(templatePrefix+"_body.html", data)
	if err != nil {
		return err
	}

	return h.sender.Send(notification.Message{To: to, Subject: subject, Body: body})
}
