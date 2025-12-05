package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/spf13/viper"
)

// Client represents a Postgres client.
type Client struct {
	pool *pgxpool.Pool
}

// Pool returns the underlying connection pool.
func (p *Client) Pool() *pgxpool.Pool {
	return p.pool
}

// Close closes the database connection for graceful shutdown.
func (p *Client) Close() {
	p.pool.Close()
}

// MustNewClient creates a new Postgres client.
func MustNewClient() *Client {
	connStr := fmt.Sprintf(
		"host=%s port=5432 user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("AUDIT_PGBOUNCER_HOST"),
		os.Getenv("AUDIT_PG_USER"),
		os.Getenv("AUDIT_PG_PASSWORD"),
		os.Getenv("AUDIT_PG_DB"),
	)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		panic(err)
	}

	if err := pool.Ping(ctx); err != nil {
		panic(err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		panic(err)
	}

	// Get a stdlib-compatible connection for goose migrations
	stdConn := pool.Config().ConnConfig.ConnString()
	gooseDB, err := goose.OpenDBWithDriver("pgx", stdConn)
	if err != nil {
		panic(err)
	}
	defer gooseDB.Close()

	if err := goose.Up(gooseDB, viper.GetString("postgres.migrations_path")); err != nil &&
		!errors.Is(err, goose.ErrNoNextVersion) {
		panic(err)
	}

	return &Client{
		pool: pool,
	}
}
