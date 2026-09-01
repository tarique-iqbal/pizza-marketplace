package notification_test

import (
	"errors"
	"os"
	"testing"

	emailapp "notification-service/internal/application/notification"
	"notification-service/internal/domain/notification"

	"github.com/stretchr/testify/assert"
)

type mockRestaurantReviewTemplateLoader struct {
	RenderCount int
	Fail        bool
	BodyData    map[string]string
}

func (m *mockRestaurantReviewTemplateLoader) Render(name string, data any) (string, error) {
	if m.Fail {
		return "", errors.New("template rendering failed")
	}
	m.RenderCount++
	switch name {
	case "restaurant_ready_for_review_subject.html":
		return "A restaurant is ready for review on MockApp", nil
	case "restaurant_ready_for_review_body.html":
		m.BodyData, _ = data.(map[string]string)
		return "Restaurant Pizza Paradise is ready for review", nil
	}
	return "", nil
}

func TestRestaurantReadyForReview_Handle_Success(t *testing.T) {
	os.Setenv("APP_NAME", "MockApp")
	os.Setenv("ADMIN_EMAIL", "admin@mock.com")

	sender := &mockSender{}
	template := &mockRestaurantReviewTemplateLoader{}
	handler := emailapp.NewRestaurantReadyForReview(sender, template)

	event := notification.EventPayload{
		Name: "restaurant.ready_for_review",
		Data: []byte(`{
			"restaurant_id": "019ff249-5c2e-76ae-8c90-117337337e66",
			"restaurant_name": "Pizza Paradise",
			"ready_at": "2026-08-11T12:00:00Z"
		}`),
	}

	err := handler.Handle(event)

	assert.NoError(t, err)
	assert.Equal(t, "admin@mock.com", sender.To)
	assert.Equal(t, "A restaurant is ready for review on MockApp", sender.Subject)
	assert.Equal(t, "Restaurant Pizza Paradise is ready for review", sender.Body)
	assert.Equal(t, 2, template.RenderCount) // subject + body
	assert.Equal(t, "Aug 11, 2026 2:00 PM CEST", template.BodyData["ready_at"])
}

func TestRestaurantReadyForReview_Handle_InvalidJSON(t *testing.T) {
	sender := &mockSender{}
	template := &mockRestaurantReviewTemplateLoader{}
	handler := emailapp.NewRestaurantReadyForReview(sender, template)

	event := notification.EventPayload{
		Name: "restaurant.ready_for_review",
		Data: []byte(`{invalid}`),
	}

	err := handler.Handle(event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character")
}

func TestRestaurantReadyForReview_Handle_TemplateRenderFails(t *testing.T) {
	sender := &mockSender{}
	template := &mockRestaurantReviewTemplateLoader{Fail: true}
	handler := emailapp.NewRestaurantReadyForReview(sender, template)

	event := notification.EventPayload{
		Name: "restaurant.ready_for_review",
		Data: []byte(`{
			"restaurant_id": "019ff249-5c2e-76ae-8c90-117337337e66",
			"restaurant_name": "Pizza Paradise",
			"ready_at": "2026-08-11T12:00:00Z"
		}`),
	}

	err := handler.Handle(event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template rendering failed")
}

func TestRestaurantReadyForReview_Handle_EmailSendFails(t *testing.T) {
	sender := &mockSender{Err: errors.New("smtp error")}
	template := &mockRestaurantReviewTemplateLoader{}
	handler := emailapp.NewRestaurantReadyForReview(sender, template)

	event := notification.EventPayload{
		Name: "restaurant.ready_for_review",
		Data: []byte(`{
			"restaurant_id": "019ff249-5c2e-76ae-8c90-117337337e66",
			"restaurant_name": "Pizza Paradise",
			"ready_at": "2026-08-11T12:00:00Z"
		}`),
	}

	err := handler.Handle(event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "smtp error")
}
