package notification_test

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	notifapp "notification-service/internal/application/notification"
	"notification-service/internal/domain/notification"
)

type recordedMessage struct {
	To      string
	Subject string
	Body    string
}

type recordingSender struct {
	messages []recordedMessage
	err      error
}

func (m *recordingSender) Send(msg notification.Message) error {
	m.messages = append(m.messages, recordedMessage{To: msg.To, Subject: msg.Subject, Body: msg.Body})

	return m.err
}

type mockOrderConfirmedTemplateLoader struct {
	fail bool
}

func (m *mockOrderConfirmedTemplateLoader) Render(name string, data any) (string, error) {
	if m.fail {
		return "", errors.New("template rendering failed")
	}

	values, _ := data.(map[string]string)

	switch name {
	case "order_confirmed_customer_subject.html":
		return "customer subject: " + values["restaurant_name"], nil
	case "order_confirmed_customer_body.html":
		return "customer body: " + values["items"], nil
	case "order_confirmed_owner_subject.html":
		return "owner subject: " + values["restaurant_name"], nil
	case "order_confirmed_owner_body.html":
		return "owner body: " + values["items"], nil
	}

	return "", nil
}

const orderConfirmedEventJSON = `{
	"order_id": "019ff249-5c2e-76ae-8c90-117337337e66",
	"restaurant_id": "019ff249-5c2e-76ae-8c90-117337337e67",
	"restaurant_name": "Pizza Paradise",
	"customer_email": "customer@example.com",
	"owner_email": "owner@pizzaparadise.de",
	"items": [{"name": "Margherita", "quantity": 2}],
	"total": "15.00",
	"currency": "EUR",
	"confirmed_at": "2026-08-11T12:00:00Z"
}`

func TestOrderConfirmed_Handle_SendsCustomerAndOwnerEmails(t *testing.T) {
	os.Setenv("APP_NAME", "MockApp")

	sender := &recordingSender{}
	template := &mockOrderConfirmedTemplateLoader{}
	handler := notifapp.NewOrderConfirmed(sender, template)

	event := notification.EventPayload{Name: "order.confirmed", Data: []byte(orderConfirmedEventJSON)}

	err := handler.Handle(event)
	require.NoError(t, err)

	require.Len(t, sender.messages, 2)

	assert.Equal(t, "customer@example.com", sender.messages[0].To)
	assert.Equal(t, "customer subject: Pizza Paradise", sender.messages[0].Subject)
	assert.Equal(t, "customer body: 2x Margherita", sender.messages[0].Body)

	assert.Equal(t, "owner@pizzaparadise.de", sender.messages[1].To)
	assert.Equal(t, "owner subject: Pizza Paradise", sender.messages[1].Subject)
	assert.Equal(t, "owner body: 2x Margherita", sender.messages[1].Body)
}

func TestOrderConfirmed_Handle_InvalidJSON(t *testing.T) {
	sender := &recordingSender{}
	template := &mockOrderConfirmedTemplateLoader{}
	handler := notifapp.NewOrderConfirmed(sender, template)

	event := notification.EventPayload{Name: "order.confirmed", Data: []byte(`{invalid}`)}

	err := handler.Handle(event)

	assert.Error(t, err)
	assert.Empty(t, sender.messages)
}

func TestOrderConfirmed_Handle_TemplateRenderFails(t *testing.T) {
	sender := &recordingSender{}
	template := &mockOrderConfirmedTemplateLoader{fail: true}
	handler := notifapp.NewOrderConfirmed(sender, template)

	event := notification.EventPayload{Name: "order.confirmed", Data: []byte(orderConfirmedEventJSON)}

	err := handler.Handle(event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template rendering failed")
}

func TestOrderConfirmed_Handle_EmailSendFails(t *testing.T) {
	sender := &recordingSender{err: errors.New("smtp error")}
	template := &mockOrderConfirmedTemplateLoader{}
	handler := notifapp.NewOrderConfirmed(sender, template)

	event := notification.EventPayload{Name: "order.confirmed", Data: []byte(orderConfirmedEventJSON)}

	err := handler.Handle(event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "smtp error")
	assert.Len(t, sender.messages, 1, "owner email must not be attempted once the customer send already failed")
}
