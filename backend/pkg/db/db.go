package db

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
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
