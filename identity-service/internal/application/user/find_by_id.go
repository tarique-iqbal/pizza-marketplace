package user

import (
	"context"

	"github.com/google/uuid"

	"identity-service/internal/domain/user"
	apperr "identity-service/internal/shared/errors"
)

type FindByID struct {
	repo user.UserRepository
}

func NewFindByID(repo user.UserRepository) *FindByID {
	return &FindByID{repo: repo}
}

func (uc *FindByID) Execute(ctx context.Context, userID uuid.UUID) (Response, error) {
	usr, err := uc.repo.FindByID(ctx, userID)
	if err != nil {
		return Response{}, err
	}

	if usr == nil {
		return Response{}, apperr.ErrNotFound
	}

	return MapToResponse(usr), nil
}
