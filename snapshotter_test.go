package pgx_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/goeventsource/goeventsource"
	"github.com/goeventsource/pgx/pgxtest"
	"github.com/goeventsource/goeventsource/goeventsourcetest/goeventsourcetestintegration"
)

func TestSnapshotter(t *testing.T) {
	if testing.Short() {
		t.Skip("This is an integration test")
	}

	factoryFunc := func(id uuid.UUID, version goeventsource.Version) *goeventsourcetestintegration.User {
		return &goeventsourcetestintegration.User{BaseRoot: goeventsource.NewBase(id, goeventsourcetestintegration.UserAggregateName, version)}
	}

	codec := goeventsource.NewJSONRootEncodeDecoder(factoryFunc)

	t.Run("always", func(t *testing.T) {
		db := pgxtest.NewUniqueSeededDatabaseConnection(t)
		strategy := goeventsource.SnapshotterWriteStrategyAlways[uuid.UUID, *goeventsourcetestintegration.User]()
		snap, err := pgxtest.NewSnapshotter(
			pgxtest.NewSnapshotterConfig(db, codec, strategy),
		)
		if err != nil {
			t.Fatal(err)
		}

		goeventsourcetestintegration.TestSnapshotterWithAlwaysStrategy(t, snap)
	})

	t.Run("never", func(t *testing.T) {
		db := pgxtest.NewUniqueSeededDatabaseConnection(t)
		strategy := goeventsource.SnapshotterWriteStrategyNever[uuid.UUID, *goeventsourcetestintegration.User]()
		snap, err := pgxtest.NewSnapshotter(
			pgxtest.NewSnapshotterConfig(db, codec, strategy),
		)
		if err != nil {
			t.Fatal(err)
		}

		goeventsourcetestintegration.TestSnapshotterWithNeverStrategy(t, snap)
	})
}
