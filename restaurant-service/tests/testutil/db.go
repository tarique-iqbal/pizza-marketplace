package testutil

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	dbinfra "restaurant-service/internal/infrastructure/db"
)

const (
	TableRestaurant  = "restaurants"
	TablePizzaSize   = "pizza_sizes"
	TableOutboxEvent = "outbox_events"
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
		postgres, err := dbinfra.NewDB()
		if err != nil {
			panic(err)
		}

		db = &TestDB{
			DB: postgres.DB,
		}
	})

	require.NotNil(t, db)

	return db
}

func (db *TestDB) TruncateTables(t *testing.T, tables ...string) {
	for _, table := range tables {
		err := db.DB.Exec(
			"TRUNCATE TABLE " + table + " RESTART IDENTITY CASCADE",
		).Error

		require.NoError(t, err)
	}
}
