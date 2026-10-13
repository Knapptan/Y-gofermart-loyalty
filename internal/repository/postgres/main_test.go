package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(60 * time.Second),
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "skip postgres tests: cannot start container: %v\n", err)
		os.Exit(0)
	}
	defer c.Terminate(ctx)

	host, _ := c.Host(ctx)
	port, _ := c.MappedPort(ctx, "5432")
	dsn := fmt.Sprintf("postgres://test:test@%s:%s/test?sslmode=disable", host, port.Port())

	testDB, err = sql.Open("postgres", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open db: %v\n", err)
		os.Exit(1)
	}
	defer testDB.Close()

	if err := applySchema(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "apply schema: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// applySchema создаёт таблицы, соответствующие migrations/000001_init.up.sql.
func applySchema(ctx context.Context) error {
	schema := `
CREATE TABLE IF NOT EXISTS users (
	id            UUID PRIMARY KEY,
	login         TEXT UNIQUE NOT NULL,
	password_hash TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS orders (
	number      TEXT PRIMARY KEY,
	user_id     UUID NOT NULL REFERENCES users(id),
	status      TEXT NOT NULL DEFAULT 'NEW',
	accrual     NUMERIC(12,2) NOT NULL DEFAULT 0,
	uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS balances (
	user_id   UUID PRIMARY KEY REFERENCES users(id),
	current   NUMERIC(12,2) NOT NULL DEFAULT 0,
	withdrawn NUMERIC(12,2) NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS withdrawals (
	id           SERIAL PRIMARY KEY,
	user_id      UUID NOT NULL REFERENCES users(id),
	order_number TEXT NOT NULL,
	sum          NUMERIC(12,2) NOT NULL,
	processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`
	_, err := testDB.ExecContext(ctx, schema)
	return err
}

// truncate очищает таблицы перед каждым тестом.
func truncate(t *testing.T, tables ...string) {
	t.Helper()
	ctx := context.Background()
	for _, tbl := range tables {
		if _, err := testDB.ExecContext(ctx, "TRUNCATE "+tbl+" CASCADE"); err != nil {
			t.Fatalf("truncate %s: %v", tbl, err)
		}
	}
}

// seedUser создаёт пользователя и возвращает его id.
func seedUser(t *testing.T, login string) string {
	t.Helper()
	repo := NewUserRepo(testDB)
	u, err := repo.Create(context.Background(), login, "$2b$dummy")
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return u.ID
}
