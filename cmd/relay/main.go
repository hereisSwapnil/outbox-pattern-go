package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"outbox-demo/internal/relay"
	"outbox-demo/pkg/db"
)

func main() {
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	connStr := fmt.Sprintf("postgres://postgres:secret@%s:5432/outboxdb", dbHost)
	replConnStr := connStr + "?replication=database"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Wait for DB to be up before starting WAL reader
	conn, err := db.ConnectWithRetry(ctx, connStr)
	if err != nil {
		log.Fatalf("db not ready: %v", err)
	}
	conn.Close(ctx)

	// Simulation for a broker publisher
	handler := func(ctx context.Context, eventType string, payload string) error {
		fmt.Printf("📨 Event received → type: %s | payload: %s\n", eventType, payload)
		// kafka.Publish(eventType, []byte(payload))
		return nil
	}

	walReader := relay.NewWALReader(replConnStr, "outbox_slot", "outbox_pub", handler)

	go func() {
		if err := walReader.Start(ctx); err != nil && err != context.Canceled {
			log.Fatalf("WAL reader stopped: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down relay...")
	cancel()
	time.Sleep(1 * time.Second)
}
