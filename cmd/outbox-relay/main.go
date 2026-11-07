package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/ThDawnWind/food-delivery-api/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/twmb/franz-go/pkg/kgo"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Outbox Relay failed: %v", err)
	}
}

func run() error {
	if err := godotenv.Load(); err != nil {
		log.Println(".env not found, using environment variables")
	}

	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		broker = "localhost:9092"
	}

	client, err := kgo.NewClient(
		kgo.SeedBrokers(broker),
	)
	if err != nil {
		return fmt.Errorf("failed to create Kafka client: %w", err)
	}
	defer client.Close()

	log.Println("Kafka client initialized")

	dbCtx, dbCancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)

	db, err := database.New(dbCtx, database.Config{
		URL: os.Getenv("DATABASE_URL"),
	})
	dbCancel()

	if err != nil {
		return fmt.Errorf("PostgreSQL unavailable: %w", err)
	}
	defer db.Close()

	shutdownCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	log.Println("Outbox Relay started")

	for shutdownCtx.Err() == nil {
		processCtx, processCancel := context.WithTimeout(
			context.Background(),
			30*time.Second,
		)

		err := processOne(processCtx, db, client)
		processCancel()

		if err != nil {
			log.Printf("Failed to process outbox event: %v", err)
		}

		select {
		case <-shutdownCtx.Done():
			log.Println("Outbox Relay shutting down...")
			return nil

		case <-time.After(3 * time.Second):
		}
	}

	log.Println("Outbox Relay shutting down...")
	return nil
}

func processOne(
	ctx context.Context,
	db *pgxpool.Pool,
	client *kgo.Client,
) error {

	var pending int

	err := db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM outbox_events
		WHERE status = 'pending'
		OR (
			status = 'processing'
			AND lease_expires_at <= NOW()
		)
	`).Scan(&pending)

	if err != nil {
		return fmt.Errorf("failed to count outbox events: %w", err)
	}

	log.Printf("Available  Outbox events: %d", pending)

	var id, orderID int64
	var eventID, eventType, payload, claimToken string

	err = db.QueryRow(ctx, `
    WITH candidate AS (
        SELECT id
        FROM outbox_events
        WHERE status = 'pending'
           OR (
               status = 'processing'
               AND lease_expires_at <= NOW()
           )
        ORDER BY id
        LIMIT 1
        FOR UPDATE SKIP LOCKED
    )
    UPDATE outbox_events AS e
    SET
        status = 'processing',
        lease_expires_at = NOW() + INTERVAL '60 seconds',
        claim_token = gen_random_uuid()
    FROM candidate
    WHERE e.id = candidate.id
    RETURNING
        e.id,
        e.event_id::text,
        e.order_id,
        e.event_type,
        e.payload::text,
        e.claim_token::text
`).Scan(
		&id,
		&eventID,
		&orderID,
		&eventType,
		&payload,
		&claimToken,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		log.Println("No available events")
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to claim outbox event: %w", err)
	}

	log.Printf("Claimed outbox event %d", id)

	log.Printf(
		"Event found: id=%d event_id=%s order_id=%d type=%s payload=%s",
		id, eventID, orderID, eventType, payload,
	)

	value, err := json.Marshal(map[string]any{
		"event_id":   eventID,
		"event_type": eventType,
		"order_id":   orderID,
		"payload":    json.RawMessage(payload),
	})
	if err != nil {
		return fmt.Errorf("failed to encode event: %w", err)
	}

	record := &kgo.Record{
		Topic: "orders.events",
		Key:   []byte(strconv.FormatInt(orderID, 10)),
		Value: value,
	}

	publishCtx, publishCancel := context.WithTimeout(
		ctx,
		10*time.Second,
	)
	defer publishCancel()

	if err := client.ProduceSync(publishCtx, record).FirstErr(); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	log.Printf("Event %s published to Kafka", eventID)

	updateCtx, updateCancel := context.WithTimeout(
		ctx,
		5*time.Second,
	)
	defer updateCancel()

	result, err := db.Exec(updateCtx, `
		UPDATE outbox_events
		SET
			status = 'published',
			published_at = NOW(),
			lease_expires_at = NULL,
			claim_token = NULL
		WHERE id = $1
		AND status = 'processing'
		AND claim_token::text = $2
		AND lease_expires_at > NOW()
	`, id, claimToken)

	if err != nil {
		return fmt.Errorf("failed to update outbox status: %w", err)
	}

	if result.RowsAffected() != 1 {
		return fmt.Errorf("outbox event %d was not updated", id)
	}

	log.Printf("Outbox event %d marked as published", id)

	return nil
}
