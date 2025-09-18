package postgres

import (
	"fmt"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/spf13/viper"
)

// Client represents a Postgres client.
type Client struct {
	db *sqlx.DB
}

// DB returns the underlying database connection.
func (p *Client) DB() *sqlx.DB {
	return p.db
}

// Close closes the database connection for graceful shutdown.
func (p *Client) Close() error {
	return p.db.Close()
}

// MustNewClient creates a new Postgres client.
func MustNewClient() *Client {
	connStr := fmt.Sprintf(
		"host=%s port=5432 user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("ORDER_PGBOUNCER_HOST"),
		os.Getenv("ORDER_PG_USER"),
		os.Getenv("ORDER_PG_PASSWORD"),
		os.Getenv("ORDER_PG_DB"),
	)
	db, err := sqlx.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}

	if err := db.Ping(); err != nil {
		panic(err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		panic(err)
	}

	if err := goose.Up(db.DB, viper.GetString("postgres.migrations_path")); err != nil {
		panic(err)
	}

	return &Client{
		db: db,
	}
}
