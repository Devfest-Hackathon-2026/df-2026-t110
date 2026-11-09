package repo

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// testDatabaseURLEnv optionally overrides the Testcontainers-managed database.
const testDatabaseURLEnv = "TEST_DATABASE_URL"

var testPostgres *postgres.PostgresContainer

// Start at most one container, only when a selected test needs a database.
// Each test still gets its own schema so parallel tests remain isolated.
var testDatabaseURL = sync.OnceValues(func() (string, error) {
	if url := os.Getenv(testDatabaseURLEnv); url != "" {
		return url, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	var err error
	testPostgres, err = postgres.Run(ctx, "postgres:18-alpine",
		postgres.WithDatabase("server_test"),
		postgres.WithUsername("server_test"),
		postgres.WithPassword("server_test"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return "", fmt.Errorf("start test PostgreSQL: %w", err)
	}
	return testPostgres.ConnectionString(ctx, "sslmode=disable")
})

func TestMain(m *testing.M) {
	code := m.Run()
	if testPostgres != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := testPostgres.Terminate(ctx)
		cancel()
		if err != nil {
			fmt.Fprintf(os.Stderr, "terminate test PostgreSQL: %v\n", err)
			code = 1
		}
	}
	os.Exit(code)
}

// newTestPool returns a pool whose search_path points at a fresh schema with
// schema.sql applied, isolating each test from the others.
func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return newTestPoolWithDDL(t, "sql/schema/schema.sql")
}

// newTestPoolWithDDL is newTestPool with another DDL file, such as the schema
// that a migration starts from.
func newTestPoolWithDDL(t *testing.T, ddlPath string) *pgxpool.Pool {
	t.Helper()
	url, err := testDatabaseURL()
	if err != nil {
		t.Fatalf("prepare test database: %v", err)
	}
	ctx := context.Background()
	ddl, err := os.ReadFile(ddlPath) //nolint:gosec // Callers pass fixed DDL paths in this package.
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}

	schema := "test_" + strings.ToLower(rand.Text())
	admin, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = admin.Close(context.Background()) })
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(context.Background(), "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE"); err != nil {
			t.Errorf("drop schema: %v", err)
		}
	})

	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, string(ddl)); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	return pool
}

func mustCreateUser(t *testing.T, pool *pgxpool.Pool, subject string, name *string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), "INSERT INTO users (google_subject, name) VALUES ($1, $2)", subject, name); err != nil {
		t.Fatalf("create user %q: %v", subject, err)
	}
}

func ptr[T any](v T) *T { return &v }

func describeName(name *string) string {
	if name == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%q", *name)
}
