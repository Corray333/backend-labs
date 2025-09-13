package postgres

import (
	"fmt"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/spf13/viper"
)

type PostgresClient struct {
	db *sqlx.DB
}

func (p *PostgresClient) DB() *sqlx.DB {
	return p.db
}

func MustNewClient() *PostgresClient {
	connStr := fmt.Sprintf("host=%s port=5432 user=%s password=%s dbname=%s sslmode=disable", os.Getenv("ORDER_PGBOUNCER_HOST"), os.Getenv("ORDER_PG_USER"), os.Getenv("ORDER_PG_PASSWORD"), os.Getenv("ORDER_PG_DB"))
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

	files, err := os.ReadDir(viper.GetString("postgres.migrations_path"))
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		fmt.Println(file.Name())
	}

	if err := goose.Up(db.DB, viper.GetString("postgres.migrations_path")); err != nil {
		panic(err)
	}

	return &PostgresClient{
		db: db,
	}
}
