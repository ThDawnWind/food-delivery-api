package main

import (
	"context"
	"errors"
	"fmt"
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

	if !errors.Is(err, errKafkaOffsetCommit) {
		t.Fatalf(
			"expected Kafka commit error marker, got %v",
			err,
		)
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

func TestPublishThenCommit_NoCommitOnPublishError(t *testing.T) {
	publishErr := errors.New("DLQ unavailable")
	commitCalled := false

	err := publishThenCommit(
		func() error {
			return publishErr
		},
		func() error {
			commitCalled = true
			return nil
		},
	)

	if !errors.Is(err, publishErr) {
		t.Fatalf("expected publish error, got %v", err)
	}

	if commitCalled {
		t.Fatal("Kafka offset must not be committed when DLQ publish fails")
	}
}

func TestPublishThenCommit_Success(t *testing.T) {
	publishCalled := false
	commitCalled := false

	err := publishThenCommit(
		func() error {
			publishCalled = true
			return nil
		},
		func() error {
			if !publishCalled {
				t.Fatal("commit called before DLQ publish")
			}

			commitCalled = true
			return nil
		},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !publishCalled {
		t.Fatal("DLQ publish was not called")
	}

	if !commitCalled {
		t.Fatal("Kafka offset commit was not called")
	}
}

func TestPublishThenCommit_CommitFailure(t *testing.T) {
	commitErr := errors.New("Kafka commit failed")

	publishCalled := false
	commitCalled := false

	err := publishThenCommit(
		func() error {
			publishCalled = true
			return nil
		},
		func() error {
			if !publishCalled {
				t.Fatal("commit called before DLQ publish")
			}

			commitCalled = true
			return commitErr
		},
	)

	if !errors.Is(err, commitErr) {
		t.Fatalf("expected commit error, got %v", err)
	}

	if !publishCalled {
		t.Fatal("DLQ publish was not called")
	}

	if !commitCalled {
		t.Fatal("Kafka offset commit was not attempted")
	}
}

func TestValidateOrderEvent(t *testing.T) {
	tests := []struct {
		name    string
		event   OrderEvent
		wantErr bool
	}{
		{
			name: "valid event",
			event: OrderEvent{
				EventID:   "550e8400-e29b-41d4-a716-446655440000",
				EventType: "OrderCreated",
				OrderID:   10,
			},
			wantErr: false,
		},
		{
			name: "empty event ID",
			event: OrderEvent{
				EventID:   "",
				EventType: "OrderCreated",
				OrderID:   10,
			},
			wantErr: true,
		},
		{
			name: "unsupported event type",
			event: OrderEvent{
				EventID:   "550e8400-e29b-41d4-a716-446655440000",
				EventType: "OrderCancelled",
				OrderID:   10,
			},
			wantErr: true,
		},
		{
			name: "invalid order ID",
			event: OrderEvent{
				EventID:   "550e8400-e29b-41d4-a716-446655440000",
				EventType: "OrderCreated",
				OrderID:   -5,
			},
			wantErr: true,
		},
		{
			name: "invalid UUID",
			event: OrderEvent{
				EventID:   "abc123",
				EventType: "OrderCreated",
				OrderID:   10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOrderEvent(tt.event)

			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"validateOrderEvent() error = %v, wantErr = %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestClassifyPostgresError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want dbErrorKind
	}{
		{
			name: "serialization failure",
			err: &pgconn.PgError{
				Code: "40001",
			},
			want: dbRetryable,
		},
		{
			name: "deadlock",
			err: &pgconn.PgError{
				Code: "40P01",
			},
			want: dbRetryable,
		},
		{
			name: "invalid text representation",
			err: &pgconn.PgError{
				Code: "22P02",
			},
			want: dbNonRetryable,
		},
		{
			name: "foreign key violation",
			err: &pgconn.PgError{
				Code: "23503",
			},
			want: dbNonRetryable,
		},
		{
			name: "check constraint violation",
			err: &pgconn.PgError{
				Code: "23514",
			},
			want: dbNonRetryable,
		},
		{
			name: "wrapped deadlock",
			err: fmt.Errorf(
				"process event: %w",
				&pgconn.PgError{Code: "40P01"},
			),
			want: dbRetryable,
		},
		{
			name: "connection error without SQLSTATE",
			err:  errors.New("connection refused"),
			want: dbUnknown,
		},
		{
			name: "unclassified SQLSTATE",
			err: &pgconn.PgError{
				Code: "23505",
			},
			want: dbUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyPostgresError(tt.err)

			if got != tt.want {
				t.Fatalf(
					"got %q, want %q",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestShouldRetryProcessing(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "no error",
			err:  nil,
			want: false,
		},
		{
			name: "serialization failure",
			err:  &pgconn.PgError{Code: "40001"},
			want: true,
		},
		{
			name: "deadlock",
			err:  &pgconn.PgError{Code: "40P01"},
			want: true,
		},
		{
			name: "foreign key violation",
			err:  &pgconn.PgError{Code: "23503"},
			want: false,
		},
		{
			name: "connection failure",
			err:  errors.New("connection refused"),
			want: true,
		},
		{
			name: "Kafka commit failure",
			err:  fmt.Errorf("%w: timeout", errKafkaOffsetCommit),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRetryProcessing(tt.err)

			if got != tt.want {
				t.Fatalf(
					"shouldRetryProcessing() = %v, want %v",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestRetryDelay(t *testing.T) {
	tests := []struct {
		attempt int
		base    time.Duration
	}{
		{0, time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{4, 16 * time.Second},
		{5, 30 * time.Second},
		{100, 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			got := retryDelay(tt.attempt)

			minDelay := tt.base - tt.base/5

			if got < minDelay || got > tt.base {
				t.Fatalf(
					"delay %v outside range [%v, %v]",
					got, minDelay, tt.base,
				)
			}
		})
	}
}
