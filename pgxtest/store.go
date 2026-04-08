package pgxtest

import (
	"testing"

	"github.com/goeventsource/goeventsource"
	"github.com/goeventsource/goeventsource/goeventsourcetest/goeventsourcetestintegration"
	"github.com/goeventsource/pgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StoreConfig represents the configuration to create a pgx.Store via NewStore
type StoreConfig struct {
	Pool           *pgxpool.Pool
	Codec          goeventsource.DomainEventEncodeDecoder
	TableName      string
	PrimaryKeyName string
}

// NewStoreConfig creates a new StoreConfig with default values for testing purposes.
func NewStoreConfig(pool *pgxpool.Pool) StoreConfig {
	return StoreConfig{
		Pool:           pool,
		Codec:          goeventsourcetestintegration.NewJSONEncodeDecoder(),
		TableName:      "store",
		PrimaryKeyName: "store_pkey",
	}
}

// NewStore creates a new instance of a pgx.Store based on the provided StoreConfig.
func NewStore[K goeventsource.ID](t *testing.T, cfg StoreConfig) *pgx.Store[K] {
	t.Helper()

	s, err := pgx.NewStore[K](cfg.Pool, cfg.Codec, cfg.TableName, cfg.PrimaryKeyName)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
