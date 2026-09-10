package persistence_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"payment-service/internal/domain/payment"
	"payment-service/internal/infrastructure/persistence"
	"payment-service/tests/infrastructure/db/fixtures"
	"payment-service/tests/testutil"
)

func setupPaymentRepo(t *testing.T) (payment.PaymentRepository, []payment.Payment) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TablePayment)

	fixturePayments := fixtures.LoadPaymentFixtures(t, db.DB)

	return persistence.NewPaymentRepository(db.DB), fixturePayments
}

func TestPaymentRepository_Create(t *testing.T) {
	db := testutil.DB(t)
	repo, _ := setupPaymentRepo(t)

	p := payment.NewPayment(
		testutil.MustNewID(),
		testutil.MustNewID(),
		testutil.MustNewID(),
		testutil.MustNewID(),
		"order",
		decimal.NewFromFloat(30.00),
		"EUR",
		decimal.NewFromFloat(2.00),
		"mollie",
	)

	err := repo.Create(context.Background(), p)
	require.NoError(t, err)

	var found payment.Payment
	err = db.DB.First(&found, "id = ?", p.ID).Error
	require.NoError(t, err)

	assert.Equal(t, p.SubjectType, found.SubjectType)
	assert.Equal(t, p.SubjectID, found.SubjectID)
	assert.True(t, p.Amount.Equal(found.Amount))
	assert.Equal(t, payment.StatusPending, found.Status)
	assert.NotZero(t, found.CreatedAt)
}

func TestPaymentRepository_Update(t *testing.T) {
	db := testutil.DB(t)
	repo, fixturePayments := setupPaymentRepo(t)

	p := fixturePayments[0]
	p.AttachGatewayReference("tr_updated")
	require.NoError(t, p.MarkSucceeded())

	err := repo.Update(context.Background(), &p)
	require.NoError(t, err)

	var found payment.Payment
	err = db.DB.First(&found, "id = ?", p.ID).Error
	require.NoError(t, err)

	assert.Equal(t, payment.StatusSucceeded, found.Status)
	require.NotNil(t, found.GatewayPaymentID)
	assert.Equal(t, "tr_updated", *found.GatewayPaymentID)
}

func TestPaymentRepository_FindByID_Found(t *testing.T) {
	repo, fixturePayments := setupPaymentRepo(t)

	target := fixturePayments[0]

	found, err := repo.FindByID(context.Background(), target.ID)
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, target.SubjectID, found.SubjectID)
}

func TestPaymentRepository_FindByID_NotFound(t *testing.T) {
	repo, _ := setupPaymentRepo(t)

	found, err := repo.FindByID(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestPaymentRepository_FindBySubject_Found(t *testing.T) {
	repo, fixturePayments := setupPaymentRepo(t)

	target := fixturePayments[0]

	found, err := repo.FindBySubject(context.Background(), target.SubjectType, target.SubjectID)
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, target.ID, found.ID)
}

func TestPaymentRepository_FindBySubject_NotFound(t *testing.T) {
	repo, _ := setupPaymentRepo(t)

	found, err := repo.FindBySubject(context.Background(), "order", uuid.New())
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestPaymentRepository_FindByGatewayPaymentID_Found(t *testing.T) {
	repo, fixturePayments := setupPaymentRepo(t)

	target := fixturePayments[1]
	require.NotNil(t, target.GatewayPaymentID)

	found, err := repo.FindByGatewayPaymentID(context.Background(), *target.GatewayPaymentID)
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, target.ID, found.ID)
}

func TestPaymentRepository_FindByGatewayPaymentID_NotFound(t *testing.T) {
	repo, _ := setupPaymentRepo(t)

	found, err := repo.FindByGatewayPaymentID(context.Background(), "tr_does_not_exist")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestPaymentRepository_WithTx_CommitsWithOuterTransaction(t *testing.T) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TablePayment)

	repo := persistence.NewPaymentRepository(db.DB)

	p := payment.NewPayment(
		testutil.MustNewID(),
		testutil.MustNewID(),
		testutil.MustNewID(),
		testutil.MustNewID(),
		"order",
		decimal.NewFromFloat(18.00),
		"EUR",
		decimal.NewFromFloat(1.20),
		"mollie",
	)

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		return repo.WithTx(tx).Create(context.Background(), p)
	})
	require.NoError(t, err)

	var found payment.Payment
	err = db.DB.First(&found, "id = ?", p.ID).Error
	require.NoError(t, err)
	assert.Equal(t, payment.StatusPending, found.Status)
}
