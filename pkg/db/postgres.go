package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
)

func ConnectWithRetry(ctx context.Context, connStr string) (*pgx.Conn, error) {
	for i := 0; i < 15; i++ {
		db, err := pgx.Connect(ctx, connStr)
		if err == nil {
			if err := db.Ping(ctx); err == nil {
				return db, nil
			}
			db.Close(ctx)
		}
		log.Printf("Waiting for Postgres (%d/15)...", i+1)
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("failed to connect to database after retries")
}
