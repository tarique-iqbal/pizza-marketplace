package commands_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	appcommands "customer-service/internal/application/address/commands"
	apperr "customer-service/internal/shared/errors"
	"customer-service/tests/testutil"
)

func TestDelete_Execute_DeletesAddress(t *testing.T) {
	addressRepo := &testutil.MockAddressRepository{}
	uc := appcommands.NewDelete(addressRepo)

	err := uc.Execute(context.Background(), testutil.MustNewID(), testutil.MustNewID())

	assert.NoError(t, err)
}

func TestDelete_Execute_PropagatesNotFound(t *testing.T) {
	addressRepo := &testutil.MockAddressRepository{DeleteErr: apperr.ErrNotFound}
	uc := appcommands.NewDelete(addressRepo)

	err := uc.Execute(context.Background(), testutil.MustNewID(), testutil.MustNewID())

	assert.ErrorIs(t, err, apperr.ErrNotFound)
}
