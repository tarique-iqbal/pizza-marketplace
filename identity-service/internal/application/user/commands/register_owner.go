package commands

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	userapp "identity-service/internal/application/user"
	"identity-service/internal/domain/auth"
	"identity-service/internal/domain/outbox"
	"identity-service/internal/domain/user"
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

func (uc *RegisterOwner) Execute(
	ctx context.Context,
	input userapp.RegisterOwnerRequest,
) (userapp.Response, error) {
	email := strings.ToLower(input.Email)

	if err := uc.emailVerifier.Verify(ctx, email, input.Code); err != nil {
		return userapp.Response{}, err
	}

	hashedPassword, err := uc.hasher.Hash(input.Password)
	if err != nil {
		return userapp.Response{}, err
	}

	userID, err := uuid.NewV7()
	if err != nil {
		return userapp.Response{}, err
	}

	restaurantID, err := uuid.NewV7()
	if err != nil {
		return userapp.Response{}, err
	}

	newUser := user.User{
		ID:        userID,
		FirstName: strings.TrimSpace(input.FirstName),
		LastName:  strings.TrimSpace(input.LastName),
		Email:     email,
		Password:  hashedPassword,
		Role:      user.RoleOwner,
		Status:    user.DefaultStatus,
	}

	newUser.MarkRegistered()
	newUser.MarkRestaurantInitiated(restaurantID, input.BusinessName, input.VATNumber)

	err = uc.db.Transaction(func(tx *gorm.DB) error {
		if err := uc.repo.WithTx(tx).Create(ctx, &newUser); err != nil {
			return err
		}

		return userapp.DispatchEventsTx(ctx, uc.outboxRepo.WithTx(tx), &newUser)
	})
	if err != nil {
		return userapp.Response{}, err
	}

	return userapp.MapToResponse(&newUser), nil
}
