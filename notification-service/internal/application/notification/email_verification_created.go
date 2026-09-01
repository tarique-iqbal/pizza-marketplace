package notification

import (
	"encoding/json"
	"notification-service/internal/domain/notification"
	"os"
)

type EmailVerificationCreated struct {
	sender   notification.Sender
	template notification.TemplateLoader
}

func NewEmailVerificationCreated(sender notification.Sender, template notification.TemplateLoader) *EmailVerificationCreated {
	return &EmailVerificationCreated{sender: sender, template: template}
}

func (h *EmailVerificationCreated) Handle(event notification.EventPayload) error {
	var payload struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return err
	}

	appName := os.Getenv("APP_NAME")
	tokenExpiryMinutes := os.Getenv("TOKEN_EXPIRY_MINUTES")

	subject, err := h.template.Render("email_verification_subject.html", map[string]string{
		"app_name": appName,
	})
	if err != nil {
		return err
	}
	body, err := h.template.Render("email_verification_body.html", map[string]string{
		"email":                payload.Email,
		"code":                 payload.Code,
		"app_name":             appName,
		"token_expiry_minutes": tokenExpiryMinutes,
	})
	if err != nil {
		return err
	}

	return h.sender.Send(notification.Message{To: payload.Email, Subject: subject, Body: body})
}
