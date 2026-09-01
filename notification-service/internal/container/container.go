package container

import (
	"os"

	notifapp "notification-service/internal/application/notification"
	"notification-service/internal/domain/notification"
	"notification-service/internal/infrastructure/messaging"
	notifemail "notification-service/internal/infrastructure/notification/email"
)

const emailTemplatePath = "internal/infrastructure/notification/email/templates"

type Container struct {
	Dispatcher notification.EventDispatcher
	Consumer   *messaging.RabbitMQConsumer
}

func NewContainer() (*Container, error) {
	amqpURL := os.Getenv("RABBITMQ_URL")
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	senderEmail := os.Getenv("SENDER_EMAIL")

	smtpSender := notifemail.NewSMTPSender(smtpHost, smtpPort, smtpUser, smtpPass, senderEmail)
	template := notifemail.NewTextTemplateLoader(emailTemplatePath)

	userRegistered := notifapp.NewUserRegistered(smtpSender, template)
	emailVerificationCreated := notifapp.NewEmailVerificationCreated(smtpSender, template)
	restaurantReadyForReview := notifapp.NewRestaurantReadyForReview(smtpSender, template)
	restaurantApproved := notifapp.NewRestaurantApproved(smtpSender, template)

	dispatcher := notifapp.NewEventDispatcher()
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
