package db

import (
	"booking-system/migrations"
	"context"
	"errors"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

var Pool *pgxpool.Pool

func Connect(ctx context.Context) {
	var err error
	Pool, err = pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("invalid DATABASE_URL: %v", err)
	}
	if err := Pool.Ping(ctx); err != nil {
		log.Fatalf("database unreachable: %v", err)
	}
}

// Migrate applies any embedded migrations the database hasn't run yet.
func Migrate() {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		log.Fatalf("migrations unreadable: %v", err)
	}
	driver, err := migratepgx.WithInstance(stdlib.OpenDBFromPool(Pool), &migratepgx.Config{})
	if err != nil {
		log.Fatalf("migration driver: %v", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "pgx5", driver)
	if err != nil {
		log.Fatalf("migration setup: %v", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("migrations failed: %v", err)
	}
}
