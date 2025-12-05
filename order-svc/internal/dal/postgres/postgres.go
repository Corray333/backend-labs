package postgres

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
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
		os.Getenv("ORDER_PGBOUNCER_HOST"),
		os.Getenv("ORDER_PG_USER"),
		os.Getenv("ORDER_PG_PASSWORD"),
		os.Getenv("ORDER_PG_DB"),
	)

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		panic(err)
	}

	// Register custom types
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		// Register v1_order composite type
		orderType, err := conn.LoadType(ctx, "v1_order")
		if err != nil {
			return fmt.Errorf("failed to load v1_order type: %w", err)
		}
		conn.TypeMap().RegisterType(orderType)

		// Register v1_order_item composite type
		orderItemType, err := conn.LoadType(ctx, "v1_order_item")
		if err != nil {
			return fmt.Errorf("failed to load v1_order_item type: %w", err)
		}
		conn.TypeMap().RegisterType(orderItemType)

		return nil
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		panic(err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		panic(err)
	}

	// Run migrations using goose with stdlib adapter
	if err := goose.SetDialect("postgres"); err != nil {
		panic(err)
	}

	db := stdlib.OpenDBFromPool(pool)
	if err := goose.Up(db, viper.GetString("postgres.migrations_path")); err != nil {
		panic(err)
	}

	return &Client{
		pool: pool,
	}
}
