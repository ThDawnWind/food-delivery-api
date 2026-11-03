package main

import (
	"context"
	"errors"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func TestRegisterEvent_RollbackOnDeliveryJobError(t *testing.T) {
	_ = godotenv.Load("../../.env")

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Fatal("TEST_DATABASE_URL is required")
	}

	parsed, err := url.Parse(dbURL)
	if err != nil || parsed.Path != "/food_delivery_test" {
		t.Fatal("test must use food_delivery_test database")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	event := OrderEvent{
		EventID:   "33333333-3333-4333-8333-333333333333",
		EventType: "OrderCreated",
		OrderID:   -1,
	}

	isNew, err := registerEvent(ctx, db, event)

	if err == nil {
		t.Fatal("expected delivery job creation error")
	}

	if isNew {
		t.Fatal("event must not be marked as processed")
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
		t.Fatalf("expected foreign key violation, got: %v", err)
	}

	var count int
	err = db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM processed_events
		WHERE consumer_name = 'delivery-jobs-consumer-v1'
		  AND event_id = $1::uuid
	`, event.EventID).Scan(&count)

	if err != nil {
		t.Fatal(err)
	}

	if count != 0 {
		t.Fatalf("expected 0 registered events, got %d", count)
	}

	t.Log("Transaction rollback verified")
}

func TestProcessAndCommit_NoCommitOnDBError(t *testing.T) {
	_ = godotenv.Load("../../.env")

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Fatal("TEST_DATABASE_URL is required")
	}

	parsed, err := url.Parse(dbURL)
	if err != nil || parsed.Path != "/food_delivery_test" {
		t.Fatal("test must use food_delivery_test database")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	event := OrderEvent{
		EventID:   "44444444-4444-4444-8444-444444444444",
		EventType: "OrderCreated",
		OrderID:   -1,
	}

	commitCalled := false

	isNew, err := processAndCommit(
		ctx,
		db,
		event,
		func(ctx context.Context) error {
			commitCalled = true
			return nil
		},
	)

	if err == nil {
		t.Fatal("expected database error")
	}

	if isNew {
		t.Fatal("event must not be marked as processed")
	}

	if commitCalled {
		t.Fatal("Kafka offset must not be committed on DB error")
	}

	var count int
	err = db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM processed_events
		WHERE consumer_name = 'delivery-jobs-consumer-v1'
		  AND event_id = $1::uuid
	`, event.EventID).Scan(&count)

	if err != nil {
		t.Fatal(err)
	}

	if count != 0 {
		t.Fatalf("expected 0 registered events, got %d", count)
	}

	t.Log("Rollback verified; Kafka offset was not committed")
}

func TestProcessAndCommit_KafkaCommitFailure(t *testing.T) {
	_ = godotenv.Load("../../.env")

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Fatal("TEST_DATABASE_URL is required")
	}

	parsed, err := url.Parse(dbURL)
	if err != nil || parsed.Path != "/food_delivery_test" {
		t.Fatal("test must use food_delivery_test database")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)

	var eventID string
	if err := db.QueryRow(ctx,
		`SELECT gen_random_uuid()::text`,
	).Scan(&eventID); err != nil {
		t.Fatal(err)
	}

	var userID, orderID int64

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(
			context.Background(), 5*time.Second,
		)
		defer cancel()

		queries := []string{
			"DELETE FROM delivery_jobs WHERE event_id = $1::uuid",
			"DELETE FROM processed_events WHERE event_id = $1::uuid",
		}
		for _, query := range queries {
			if _, err := db.Exec(cleanupCtx, query, eventID); err != nil {
				t.Errorf("cleanup failed: %v", err)
			}
		}

		if orderID != 0 {
			if _, err := db.Exec(cleanupCtx,
				"DELETE FROM orders WHERE id = $1", orderID,
			); err != nil {
				t.Errorf("cleanup order: %v", err)
			}
		}
		if userID != 0 {
			if _, err := db.Exec(cleanupCtx,
				"DELETE FROM users WHERE id = $1", userID,
			); err != nil {
				t.Errorf("cleanup user: %v", err)
			}
		}
	})

	err = db.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash)
		VALUES (
			'cc-' || $1::text,
			'cc-' || $1::text || '@example.com',
			'test-hash'
		)
		RETURNING id
	`, eventID).Scan(&userID)
	if err != nil {
		t.Fatal(err)
	}

	err = db.QueryRow(ctx, `
		INSERT INTO orders (user_id, total_price, delivery_address)
		VALUES ($1, 100, 'Test address')
		RETURNING id
	`, userID).Scan(&orderID)
	if err != nil {
		t.Fatal(err)
	}

	event := OrderEvent{
		EventID:   eventID,
		EventType: "OrderCreated",
		OrderID:   orderID,
	}

	isNew, err := processAndCommit(
		ctx, db, event,
		func(context.Context) error {
			return errors.New("simulated Kafka commit failure")
		},
	)

	if err == nil {
		t.Fatal("expected Kafka commit error")
	}
	if !isNew {
		t.Fatal("expected newly processed event")
	}

	var events, jobs int
	err = db.QueryRow(ctx, `
		SELECT COUNT(*) FROM processed_events
		WHERE consumer_name = 'delivery-jobs-consumer-v1'
		  AND event_id = $1::uuid
	`, eventID).Scan(&events)
	if err != nil {
		t.Fatal(err)
	}

	err = db.QueryRow(ctx, `
		SELECT COUNT(*) FROM delivery_jobs
		WHERE event_id = $1::uuid
	`, eventID).Scan(&jobs)
	if err != nil {
		t.Fatal(err)
	}

	if events != 1 || jobs != 1 {
		t.Fatalf("expected 1 event and 1 job, got %d and %d",
			events, jobs)
	}

	commitCalled := false

	isNew, err = processAndCommit(
		ctx,
		db,
		event,
		func(context.Context) error {
			commitCalled = true
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if isNew {
		t.Fatal("duplicate must not be processed again")
	}

	if !commitCalled {
		t.Fatal("Kafka offset must be committed for duplicate event")
	}

	var jobsAfterRetry int

	err = db.QueryRow(ctx, `
    SELECT COUNT(*)
    FROM delivery_jobs
    WHERE event_id = $1::uuid
`, eventID).Scan(&jobsAfterRetry)

	if err != nil {
		t.Fatal(err)
	}

	if jobsAfterRetry != 1 {
		t.Fatalf(
			"expected 1 delivery job after retry, got %d",
			jobsAfterRetry,
		)
	}

	t.Log("Duplicate delivery prevented")

	t.Log("PostgreSQL committed; retry safely skipped duplicate")
}
