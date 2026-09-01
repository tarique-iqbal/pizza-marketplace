package notification

import (
	"encoding/json"
	"fmt"
	"notification-service/internal/domain/notification"
	"os"
)

const (
	RoleCustomer = "customer"
	RoleOwner    = "owner"
)

type UserRegistered struct {
	sender   notification.Sender
	template notification.TemplateLoader
}

func NewUserRegistered(sender notification.Sender, template notification.TemplateLoader) *UserRegistered {
	return &UserRegistered{sender: sender, template: template}
}

func (h *UserRegistered) Handle(event notification.EventPayload) error {
	var payload struct {
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		Role      string `json:"role"`
		Timestamp string `json:"timestamp"`
	}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return err
	}

	validRoles := map[string]bool{RoleCustomer: true, RoleOwner: true}
	if !validRoles[payload.Role] {
		return fmt.Errorf("invalid role in event payload: %s", payload.Role)
	}

	subjectTemplate := fmt.Sprintf("%s_welcome_email_subject.html", payload.Role)
	bodyTemplate := fmt.Sprintf("%s_welcome_email_body.html", payload.Role)

	appName := os.Getenv("APP_NAME")
	supportEmail := os.Getenv("SUPPORT_EMAIL")

	subject, err := h.template.Render(subjectTemplate, map[string]string{
		"app_name": appName,
	})
	if err != nil {
		return err
	}

	body, err := h.template.Render(bodyTemplate, map[string]string{
		"first_name":    payload.FirstName,
		"app_name":      appName,
		"support_email": supportEmail,
	})
	if err != nil {
		return err
	}

	return h.sender.SendEmail(payload.Email, subject, body)
}
