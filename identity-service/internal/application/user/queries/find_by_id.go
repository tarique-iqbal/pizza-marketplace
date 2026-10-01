package queries

import (
	"context"

	"github.com/google/uuid"

	userapp "identity-service/internal/application/user"
	"identity-service/internal/domain/user"
	apperr "identity-service/internal/shared/errors"
)

type FindByID struct {
	repo user.UserRepository
}

func NewFindByID(repo user.UserRepository) *FindByID {
	return &FindByID{repo: repo}
}

func (qry *FindByID) Execute(ctx context.Context, userID uuid.UUID) (userapp.Response, error) {
	usr, err := qry.repo.FindByID(ctx, userID)
	if err != nil {
		return userapp.Response{}, err
	}

	if usr == nil {
		return userapp.Response{}, apperr.ErrNotFound
	}

	return userapp.MapToResponse(usr), nil
}
