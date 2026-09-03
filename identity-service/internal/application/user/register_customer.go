package user

import (
	"context"
	"encoding/json"
	"identity-service/internal/domain/auth"
	"identity-service/internal/domain/outbox"
	"identity-service/internal/domain/user"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RegisterCustomer struct {
	db            *gorm.DB
	emailVerifier auth.EmailVerifier
	repo          user.UserRepository
	hasher        auth.PasswordHasher
	outboxRepo    outbox.OutboxRepository
}

func NewRegisterCustomer(
	db *gorm.DB,
	emailVerifier auth.EmailVerifier,
	repo user.UserRepository,
	hasher auth.PasswordHasher,
	outboxRepo outbox.OutboxRepository,
) *RegisterCustomer {
	return &RegisterCustomer{
		db:            db,
		emailVerifier: emailVerifier,
		repo:          repo,
		hasher:        hasher,
		outboxRepo:    outboxRepo,
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

	userRegistered := UserRegistered{
		UserID:     newUser.ID,
		Email:      newUser.Email,
		FirstName:  newUser.FirstName,
		Role:       newUser.Role,
		OccurredAt: time.Now().UTC(),
	}
	userRegistered.EventName = userRegistered.GetEventName()

	payload, err := json.Marshal(userRegistered)
	if err != nil {
		return Response{}, err
	}

	err = uc.db.Transaction(func(tx *gorm.DB) error {
		if err := uc.repo.WithTx(tx).Create(ctx, &newUser); err != nil {
			return err
		}

		outboxEvent := outbox.NewOutboxEvent(newUser.ID, outbox.EventUserRegistered, payload)
		return uc.outboxRepo.WithTx(tx).Create(ctx, &outboxEvent)
	})
	if err != nil {
		return Response{}, err
	}

	return MapToResponse(&newUser), nil
}
