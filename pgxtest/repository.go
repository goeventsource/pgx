package pgxtest

import (
	"github.com/goeventsource/goeventsource"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/goeventsource/pgx"
)

// RepositoryConfig represents the configuration to create a pgx.Repository via NewRepository
type RepositoryConfig[K goeventsource.ID, V goeventsource.Root[K]] struct {
	StoreConfig
	FactoryFunc goeventsource.FactoryFunc[K, V]
	Opts        []pgx.RepositoryOpt[K, V]
}

// NewRepositoryConfig creates a new RepositoryConfig with default values for testing purposes.
func NewRepositoryConfig[K goeventsource.ID, V goeventsource.Root[K]](
	pool *pgxpool.Pool,
	factoryFunc goeventsource.FactoryFunc[K, V],
	opts ...pgx.RepositoryOpt[K, V],
) RepositoryConfig[K, V] {
	return RepositoryConfig[K, V]{
		StoreConfig: NewStoreConfig(pool),
		FactoryFunc: factoryFunc,
		Opts:        opts,
	}
}

// NewRepository creates a new instance of a pgx.Repository based on the provided StoreConfig.
func NewRepository[K goeventsource.ID, V goeventsource.Root[K]](cfg RepositoryConfig[K, V]) (*pgx.Repository[K, V], *pgx.Store[K], error) {
	s, err := pgx.NewStore[K](cfg.Pool, cfg.Codec, cfg.TableName, cfg.PrimaryKeyName)
	if err != nil {
		return nil, nil, err
	}
	return pgx.NewRepository(cfg.Pool, s, cfg.FactoryFunc, cfg.Opts...), s, nil
}
