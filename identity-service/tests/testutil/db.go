package testutil

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	dbinfra "identity-service/internal/infrastructure/db"
)

const (
	TableEmailVerification = "email_verifications"
	TableUser              = "users"
	TableOutboxEvent       = "outbox_events"
)

type TestDB struct {
	DB *gorm.DB
}

var (
	dbOnce sync.Once
	db     *TestDB
)

func DB(t *testing.T) *TestDB {
	dbOnce.Do(func() {
		tdb, err := dbinfra.NewDB()
		if err != nil {
			panic(err)
		}

		db = &TestDB{
			DB: tdb.DB,
		}
	})

	require.NotNil(t, db)

	return db
}

func (tdb *TestDB) TruncateTables(t *testing.T, tables ...string) {
	for _, table := range tables {
		err := tdb.DB.Exec(
			"TRUNCATE TABLE " + table + " RESTART IDENTITY CASCADE",
		).Error

		require.NoError(t, err)
	}
}
