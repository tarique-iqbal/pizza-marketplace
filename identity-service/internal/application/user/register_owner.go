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

type RegisterOwner struct {
	db            *gorm.DB
	emailVerifier auth.EmailVerifier
	hasher        auth.PasswordHasher
	repo          user.UserRepository
	outboxRepo    outbox.OutboxRepository
}

func NewRegisterOwner(
	db *gorm.DB,
	emailVerifier auth.EmailVerifier,
	hasher auth.PasswordHasher,
	repo user.UserRepository,
	outboxRepo outbox.OutboxRepository,
) *RegisterOwner {
	return &RegisterOwner{
		db:            db,
		emailVerifier: emailVerifier,
		hasher:        hasher,
		repo:          repo,
		outboxRepo:    outboxRepo,
	}
}

func (uc *RegisterOwner) Execute(ctx context.Context, input RegisterOwnerRequest) (Response, error) {
	if err := uc.emailVerifier.Verify(ctx, input.Email, input.Code); err != nil {
		return Response{}, err
	}

	hashedPassword, err := uc.hasher.Hash(input.Password)
	if err != nil {
		return Response{}, err
	}

	userID, err := uuid.NewV7()
	if err != nil {
		return Response{}, err
	}

	restaurantID, err := uuid.NewV7()
	if err != nil {
		return Response{}, err
	}

	newUser := user.User{
		ID:        userID,
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Email:     input.Email,
		Password:  hashedPassword,
		Role:      user.RoleOwner,
		Status:    user.DefaultStatus,
	}

	restaurantInitiatedPayload, err := json.Marshal(map[string]interface{}{
		"restaurant_id": restaurantID,
		"owner_id":      userID,
		"business_name": input.BusinessName,
		"vat_number":    input.VATNumber,
	})
	if err != nil {
		return Response{}, err
	}

	userRegistered := UserRegistered{
		Email:      newUser.Email,
		FirstName:  newUser.FirstName,
		Role:       newUser.Role,
		OccurredAt: time.Now().UTC(),
	}
	userRegistered.EventName = userRegistered.GetEventName()

	userRegisteredPayload, err := json.Marshal(userRegistered)
	if err != nil {
		return Response{}, err
	}

	err = uc.db.Transaction(func(tx *gorm.DB) error {
		if err := uc.repo.WithTx(tx).Create(ctx, &newUser); err != nil {
			return err
		}

		restaurantInitiated := outbox.NewOutboxEvent(
			restaurantID,
			outbox.EventRestaurantInitiated,
			restaurantInitiatedPayload,
		)
		if err := uc.outboxRepo.WithTx(tx).Create(ctx, &restaurantInitiated); err != nil {
			return err
		}

		userRegisteredEvent := outbox.NewOutboxEvent(userID, outbox.EventUserRegistered, userRegisteredPayload)
		return uc.outboxRepo.WithTx(tx).Create(ctx, &userRegisteredEvent)
	})
	if err != nil {
		return Response{}, err
	}

	return MapToResponse(&newUser), nil
}
