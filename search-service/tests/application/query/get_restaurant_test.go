package query_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"search-service/internal/application/query"
	"search-service/internal/domain/index"
	apperr "search-service/internal/shared/errors"
	"search-service/tests/testutil"
)

func TestGetRestaurant_Found_ReturnsIndexedRestaurant(t *testing.T) {
	id := uuid.New()
	repo := &testutil.MockSearchRepository{
		FindByIDResult: index.IndexedRestaurant{ID: id, Name: "Anatolische Kueche"},
	}
	uc := query.NewGetRestaurant(repo)

	result, err := uc.Execute(context.Background(), id)
	require.NoError(t, err)

	assert.Equal(t, "Anatolische Kueche", result.Name)
}

func TestGetRestaurant_NotFound_ReturnsErrNotFound(t *testing.T) {
	id := uuid.New()
	repo := &testutil.MockSearchRepository{
		FindByIDErr: fmt.Errorf("restaurant %s: %w", id, apperr.ErrNotFound),
	}
	uc := query.NewGetRestaurant(repo)

	_, err := uc.Execute(context.Background(), id)

	require.Error(t, err)
	assert.True(t, errors.Is(err, apperr.ErrNotFound))
}

func TestGetRestaurant_RepositoryError_PropagatesError(t *testing.T) {
	id := uuid.New()
	repo := &testutil.MockSearchRepository{
		FindByIDErr: errors.New("es unreachable"),
	}
	uc := query.NewGetRestaurant(repo)

	_, err := uc.Execute(context.Background(), id)

	require.Error(t, err)
	assert.False(t, errors.Is(err, apperr.ErrNotFound))
}
