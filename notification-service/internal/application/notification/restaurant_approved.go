package notification

import (
	"encoding/json"
	"notification-service/internal/domain/notification"
	"os"
	"time"
	_ "time/tzdata"
)

type RestaurantApproved struct {
	sender   notification.Sender
	template notification.TemplateLoader
}

func NewRestaurantApproved(
	sender notification.Sender,
	template notification.TemplateLoader,
) *RestaurantApproved {
	return &RestaurantApproved{sender: sender, template: template}
}

func (h *RestaurantApproved) Handle(event notification.EventPayload) error {
	var payload struct {
		RestaurantID   string    `json:"restaurant_id"`
		RestaurantName string    `json:"restaurant_name"`
		Email          string    `json:"email"`
		ApprovedAt     time.Time `json:"approved_at"`
	}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return err
	}

	appName := os.Getenv("APP_NAME")

	approvedAt := payload.ApprovedAt
	if loc, err := time.LoadLocation("Europe/Berlin"); err == nil {
		approvedAt = approvedAt.In(loc)
	}

	subject, err := h.template.Render("restaurant_approved_subject.html", map[string]string{
		"restaurant_name": payload.RestaurantName,
		"app_name":        appName,
	})
	if err != nil {
		return err
	}

	body, err := h.template.Render("restaurant_approved_body.html", map[string]string{
		"restaurant_name": payload.RestaurantName,
		"approved_at":     approvedAt.Format("Jan 2, 2006 3:04 PM MST"),
		"app_name":        appName,
	})
	if err != nil {
		return err
	}

	return h.sender.Send(notification.Message{To: payload.Email, Subject: subject, Body: body})
}
