package pgxtest

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	//go:embed testdata/event_store_table.sql
	eventStoreTableMigration string
	//go:embed testdata/snapshot_table.sql
	snapshotTableMigration string

	mu sync.Mutex

	sharedPool      *pgxpool.Pool
	sharedContainer testcontainers.Container
)

func newPoolAndContainer(ctx context.Context) (*pgxpool.Pool, testcontainers.Container, error) {
	ctr, err := testcontainers.GenericContainer(
		ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:          "postgres:13-alpine",
				ExposedPorts:   []string{"5432/tcp"},
				NetworkAliases: map[string][]string{"5432": {"5432"}},
				WaitingFor: (&wait.LogStrategy{
					Log:          "database system is ready to accept connections",
					Occurrence:   2,
					PollInterval: 100 * time.Millisecond,
				}).WithStartupTimeout(time.Minute),
				Env: map[string]string{
					"POSTGRES_DB":       "source",
					"POSTGRES_PASSWORD": "source",
					"POSTGRES_USER":     "source",
				},
			},
			Started: true,
			Logger:  log.Default(),
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("database container was not started: %w", err)
	}

	host, err := ctr.Host(ctx)
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, nil, fmt.Errorf("database container host: %w", err)
	}

	port, err := ctr.MappedPort(ctx, "5432")
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, nil, fmt.Errorf("database container port: %w", err)
	}

	url := fmt.Sprintf("postgres://source:source@%v:%v/source?sslmode=disable", host, port.Port())

	p, err := pgxpool.New(ctx, url)
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, nil, fmt.Errorf("pgx pool: %w", err)
	}

	if _, err := p.Exec(ctx, eventStoreTableMigration); err != nil {
		p.Close()
		_ = ctr.Terminate(ctx)
		return nil, nil, fmt.Errorf("event store migration: %w", err)
	}

	if _, err := p.Exec(ctx, snapshotTableMigration); err != nil {
		p.Close()
		_ = ctr.Terminate(ctx)
		return nil, nil, fmt.Errorf("snapshot migration: %w", err)
	}

	return p, ctr, nil
}

// NewUniqueSeededDatabaseConnection creates or reuse a seeded database connection for testing purposes.
// It starts a PostgreSQL container, performs necessary setup, and returns the connection pool.
// It is recommended to call CleanUp function in a TestMain after all the tests were executed
func NewUniqueSeededDatabaseConnection(t *testing.T) *pgxpool.Pool {
	mu.Lock()
	defer mu.Unlock()

	ctx := context.Background()

	if sharedPool != nil {
		return sharedPool
	}

	p, ctr, err := newPoolAndContainer(ctx)
	if err != nil {
		t.Fatalf("%v", err)
	}
	sharedPool, sharedContainer = p, ctr
	return sharedPool
}

// NewSeededDatabaseConnection creates a new seeded database connection for testing purposes.
// It starts a PostgreSQL container, performs necessary setup, and returns the connection pool.
func NewSeededDatabaseConnection(t *testing.T) *pgxpool.Pool {
	ctx := context.Background()

	p, ctr, err := newPoolAndContainer(ctx)
	if err != nil {
		t.Fatalf("%v", err)
	}

	t.Cleanup(func() {
		if err := ctr.Terminate(ctx); err != nil {
			t.Errorf("database container was not closed: %s", err)
		}
	})

	t.Cleanup(func() {
		p.Close()
	})

	return p
}

// NewDemoPool starts a postgres testcontainer and returns a pool plus a cleanup func (for examples / cmd).
func NewDemoPool(ctx context.Context) (*pgxpool.Pool, func(), error) {
	p, ctr, err := newPoolAndContainer(ctx)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		p.Close()
		if err := ctr.Terminate(ctx); err != nil {
			log.Printf("terminate container: %v", err)
		}
	}
	return p, cleanup, nil
}

func CleanUp() {
	ctx := context.Background()

	if sharedPool == nil {
		return
	}

	sharedPool.Close()

	if err := sharedContainer.Terminate(ctx); err != nil {
		log.Fatalf("database container was not closed: %s", err)
	}
}
