package persistence

import (
	"context"
	"identity-service/internal/domain/auth"

	"gorm.io/gorm"
)

type emailVerificationRepo struct {
	db *gorm.DB
}

func NewEmailVerificationRepository(db *gorm.DB) auth.EmailVerificationRepository {
	return &emailVerificationRepo{db: db}
}

func (repo *emailVerificationRepo) WithTx(tx *gorm.DB) auth.EmailVerificationRepository {
	return &emailVerificationRepo{db: tx}
}

func (repo *emailVerificationRepo) FindByEmail(
	ctx context.Context,
	email string,
) (*auth.EmailVerification, error) {
	var ev auth.EmailVerification
	err := repo.db.Where("email = ?", email).First(&ev).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &ev, nil
}

func (repo *emailVerificationRepo) Create(
	ctx context.Context,
	ev *auth.EmailVerification,
) error {
	return repo.db.Create(ev).Error
}

func (repo *emailVerificationRepo) Updates(
	ctx context.Context,
	ev *auth.EmailVerification,
) error {
	return repo.db.Updates(ev).Error
}
