package container

import (
	"os"

	emailapp "email-service/internal/application/email"
	"email-service/internal/domain/email"
	emailinfra "email-service/internal/infrastructure/email"
	"email-service/internal/infrastructure/messaging"
)

const emailTemplatePath = "internal/infrastructure/email/templates"

type Container struct {
	Dispatcher email.EventDispatcher
	Consumer   *messaging.RabbitMQConsumer
}

func NewContainer() (*Container, error) {
	amqpURL := os.Getenv("RABBITMQ_URL")
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	senderEmail := os.Getenv("SENDER_EMAIL")

	smtpSender := emailinfra.NewSMTPSender(smtpHost, smtpPort, smtpUser, smtpPass, senderEmail)
	template := emailinfra.NewTextTemplateLoader(emailTemplatePath)

	userRegistered := emailapp.NewUserRegistered(smtpSender, template)
	emailVerificationCreated := emailapp.NewEmailVerificationCreated(smtpSender, template)
	restaurantReadyForReview := emailapp.NewRestaurantReadyForReview(smtpSender, template)
	restaurantApproved := emailapp.NewRestaurantApproved(smtpSender, template)

	dispatcher := emailapp.NewEventDispatcher()
	dispatcher.Register(messaging.Exchanges["identity.events"][1], userRegistered)
	dispatcher.Register(messaging.Exchanges["identity.events"][0], emailVerificationCreated)
	dispatcher.Register(messaging.Exchanges["restaurant.events"][0], restaurantReadyForReview)
	dispatcher.Register(messaging.Exchanges["restaurant.events"][1], restaurantApproved)

	consumer, err := messaging.NewRabbitMQConsumer(amqpURL)

	return &Container{
		Dispatcher: dispatcher,
		Consumer:   consumer,
	}, err
}
