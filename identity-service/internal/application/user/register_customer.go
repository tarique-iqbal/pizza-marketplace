package user

import (
	"context"
	"identity-service/internal/domain/auth"
	"identity-service/internal/domain/user"
	logobs "identity-service/internal/infrastructure/observability/logger"
	"identity-service/internal/shared/event"
	"time"

	"github.com/google/uuid"
)

type RegisterCustomer struct {
	emailVerifier auth.EmailVerifier
	repo          user.UserRepository
	hasher        auth.PasswordHasher
	publisher     event.EventPublisher
}

func NewRegisterCustomer(
	emailVerifier auth.EmailVerifier,
	repo user.UserRepository,
	hasher auth.PasswordHasher,
	publisher event.EventPublisher,
) *RegisterCustomer {
	return &RegisterCustomer{
		emailVerifier: emailVerifier,
		repo:          repo,
		hasher:        hasher,
		publisher:     publisher,
	}
}

func (uc *RegisterCustomer) Execute(ctx context.Context, input RegisterCustomerRequest) (Response, error) {
	if err := uc.emailVerifier.Verify(ctx, input.Email, input.Code); err != nil {
		return Response{}, err
	}

	hashedPassword, err := uc.hasher.Hash(input.Password)
	if err != nil {
		return Response{}, err
	}

	newUser := user.User{
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Email:     input.Email,
		Password:  hashedPassword,
		Role:      user.RoleCustomer,
		Status:    user.DefaultStatus,
	}

	userID, err := uuid.NewV7()
	if err != nil {
		return Response{}, err
	}
	newUser.ID = userID

	if err := uc.repo.Create(ctx, &newUser); err != nil {
		return Response{}, err
	}

	event := UserRegistered{
		Email:     newUser.Email,
		FirstName: newUser.FirstName,
		Role:      newUser.Role,
		Timestamp: time.Now().UTC(),
	}
	event.EventName = event.GetEventName()

	if err := uc.publisher.PublishEvent(ctx, event); err != nil {
		logobs.FromContext(ctx).Warn("failed to publish user.registered event", "error", err)
	}

	return MapToResponse(&newUser), nil
}
