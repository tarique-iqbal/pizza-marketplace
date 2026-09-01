package notification_test

import (
	"errors"
	"os"
	"testing"

	notifapp "notification-service/internal/application/notification"
	"notification-service/internal/domain/notification"

	"github.com/stretchr/testify/assert"
)

type mockRestaurantApprovedTemplateLoader struct {
	RenderCount int
	Fail        bool
	BodyData    map[string]string
}

func (m *mockRestaurantApprovedTemplateLoader) Render(name string, data any) (string, error) {
	if m.Fail {
		return "", errors.New("template rendering failed")
	}
	m.RenderCount++
	switch name {
	case "restaurant_approved_subject.html":
		return "Your restaurant \"Pizza Paradise\" has been approved on MockApp", nil
	case "restaurant_approved_body.html":
		m.BodyData, _ = data.(map[string]string)
		return "Restaurant Pizza Paradise has been approved", nil
	}
	return "", nil
}

func TestRestaurantApproved_Handle_Success(t *testing.T) {
	os.Setenv("APP_NAME", "MockApp")

	sender := &mockSender{}
	template := &mockRestaurantApprovedTemplateLoader{}
	handler := notifapp.NewRestaurantApproved(sender, template)

	event := notification.EventPayload{
		Name: "restaurant.approved",
		Data: []byte(`{
			"restaurant_id": "019ff249-5c2e-76ae-8c90-117337337e66",
			"restaurant_name": "Pizza Paradise",
			"email": "kontakt@pizzaparadise.de",
			"approved_at": "2026-08-11T12:00:00Z"
		}`),
	}

	err := handler.Handle(event)

	assert.NoError(t, err)
	assert.Equal(t, "kontakt@pizzaparadise.de", sender.To)
	assert.Equal(t, "Your restaurant \"Pizza Paradise\" has been approved on MockApp", sender.Subject)
	assert.Equal(t, "Restaurant Pizza Paradise has been approved", sender.Body)
	assert.Equal(t, 2, template.RenderCount) // subject + body
	assert.Equal(t, "Aug 11, 2026 2:00 PM CEST", template.BodyData["approved_at"])
}

func TestRestaurantApproved_Handle_InvalidJSON(t *testing.T) {
	sender := &mockSender{}
	template := &mockRestaurantApprovedTemplateLoader{}
	handler := notifapp.NewRestaurantApproved(sender, template)

	event := notification.EventPayload{
		Name: "restaurant.approved",
		Data: []byte(`{invalid}`),
	}

	err := handler.Handle(event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character")
}

func TestRestaurantApproved_Handle_TemplateRenderFails(t *testing.T) {
	sender := &mockSender{}
	template := &mockRestaurantApprovedTemplateLoader{Fail: true}
	handler := notifapp.NewRestaurantApproved(sender, template)

	event := notification.EventPayload{
		Name: "restaurant.approved",
		Data: []byte(`{
			"restaurant_id": "019ff249-5c2e-76ae-8c90-117337337e66",
			"restaurant_name": "Pizza Paradise",
			"email": "kontakt@pizzaparadise.de",
			"approved_at": "2026-08-11T12:00:00Z"
		}`),
	}

	err := handler.Handle(event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template rendering failed")
}

func TestRestaurantApproved_Handle_EmailSendFails(t *testing.T) {
	sender := &mockSender{Err: errors.New("smtp error")}
	template := &mockRestaurantApprovedTemplateLoader{}
	handler := notifapp.NewRestaurantApproved(sender, template)

	event := notification.EventPayload{
		Name: "restaurant.approved",
		Data: []byte(`{
			"restaurant_id": "019ff249-5c2e-76ae-8c90-117337337e66",
			"restaurant_name": "Pizza Paradise",
			"email": "kontakt@pizzaparadise.de",
			"approved_at": "2026-08-11T12:00:00Z"
		}`),
	}

	err := handler.Handle(event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "smtp error")
}
