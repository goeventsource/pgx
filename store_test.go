package pgx_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/goeventsource/goeventsource/goeventsourcetest/goeventsourcetestintegration"

	"github.com/goeventsource/pgx/pgxtest"
)

func TestStore(t *testing.T) {
	if testing.Short() {
		t.Skip("This is an integration test")
	}

	db := pgxtest.NewUniqueSeededDatabaseConnection(t)
	store := pgxtest.NewStore[uuid.UUID](t, pgxtest.NewStoreConfig(db))
	goeventsourcetestintegration.TestStore(t, store)
}
