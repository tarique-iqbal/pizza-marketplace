package email

import (
	"email-service/internal/domain/email"
	"encoding/json"
	"os"
	"time"
	_ "time/tzdata"
)

type RestaurantReadyForReview struct {
	sender   email.Sender
	template email.TemplateLoader
}

func NewRestaurantReadyForReview(
	sender email.Sender,
	template email.TemplateLoader,
) *RestaurantReadyForReview {
	return &RestaurantReadyForReview{sender: sender, template: template}
}

func (h *RestaurantReadyForReview) Handle(event email.EventPayload) error {
	var payload struct {
		RestaurantID   string    `json:"restaurant_id"`
		RestaurantName string    `json:"restaurant_name"`
		ReadyAt        time.Time `json:"ready_at"`
	}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return err
	}

	appName := os.Getenv("APP_NAME")
	adminEmail := os.Getenv("ADMIN_EMAIL")

	readyAt := payload.ReadyAt
	if loc, err := time.LoadLocation("Europe/Berlin"); err == nil {
		readyAt = readyAt.In(loc)
	}

	subject, err := h.template.Render("restaurant_ready_for_review_subject.html", map[string]string{
		"app_name": appName,
	})
	if err != nil {
		return err
	}

	body, err := h.template.Render("restaurant_ready_for_review_body.html", map[string]string{
		"restaurant_id":   payload.RestaurantID,
		"restaurant_name": payload.RestaurantName,
		"ready_at":        readyAt.Format("Jan 2, 2006 3:04 PM MST"),
		"app_name":        appName,
	})
	if err != nil {
		return err
	}

	return h.sender.SendEmail(adminEmail, subject, body)
}
