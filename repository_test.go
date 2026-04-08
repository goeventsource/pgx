package pgx_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/goeventsource/goeventsource"
	"github.com/goeventsource/goeventsource/goeventsourcetest/goeventsourcetestintegration"

	"github.com/goeventsource/pgx"
	"github.com/goeventsource/pgx/pgxtest"
)

func TestRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("This is an integration test")
	}

	db := pgxtest.NewUniqueSeededDatabaseConnection(t)
	factoryFunc := func(id uuid.UUID, version goeventsource.Version) *goeventsourcetestintegration.User {
		return &goeventsourcetestintegration.User{BaseRoot: goeventsource.NewBase(id, goeventsourcetestintegration.UserAggregateName, version)}
	}

	t.Run("repository", func(t *testing.T) {
		r, s, err := pgxtest.NewRepository(pgxtest.NewRepositoryConfig(db, factoryFunc))
		if err != nil {
			t.Fatal(err)
		}
		goeventsourcetestintegration.TestRepository(t, r, s)
	})

	t.Run("repository_with_snapshots", func(t *testing.T) {
		strategy := goeventsource.SnapshotterWriteStrategyAlways[uuid.UUID, *goeventsourcetestintegration.User]()
		codec := goeventsource.NewJSONRootEncodeDecoder(factoryFunc)
		snap, err := pgxtest.NewSnapshotter(
			pgxtest.NewSnapshotterConfig(db, codec, strategy),
		)
		if err != nil {
			t.Fatal(err)
		}

		r, s, err := pgxtest.NewRepository(
			pgxtest.NewRepositoryConfig(
				db,
				factoryFunc,
				pgx.WithSnapshotterOpt(snap),
			),
		)
		if err != nil {
			t.Fatal(err)
		}
		goeventsourcetestintegration.TestRepository(t, r, s)
	})

	t.Run("repository_with_projector", func(t *testing.T) {
		proj := &goeventsourcetestintegration.Projector{}

		r, s, err := pgxtest.NewRepository(
			pgxtest.NewRepositoryConfig(
				db,
				factoryFunc,
				pgx.WithProjectorOpt[uuid.UUID, *goeventsourcetestintegration.User](proj),
			),
		)
		if err != nil {
			t.Fatal(err)
		}
		goeventsourcetestintegration.TestRepositoryWithProjector(t, r, proj, s)
	})
}
