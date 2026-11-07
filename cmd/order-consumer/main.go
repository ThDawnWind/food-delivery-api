package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/ThDawnWind/food-delivery-api/internal/database"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/twmb/franz-go/pkg/kgo"
)

type OrderEvent struct {
	EventID   string          `json:"event_id"`
	EventType string          `json:"event_type"`
	OrderID   int64           `json:"order_id"`
	Payload   json.RawMessage `json:"payload"`
}

var errKafkaOffsetCommit = errors.New("kafka offset commit failed")

var uuidPattern = regexp.MustCompile(
	`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,
)

func validateOrderEvent(event OrderEvent) error {
	if event.EventID == "" {
		return fmt.Errorf("event_id is required")
	}

	if !uuidPattern.MatchString(event.EventID) {
		return fmt.Errorf(
			"invalid event_id UUID: %q",
			event.EventID,
		)
	}

	if event.EventType != "OrderCreated" {
		return fmt.Errorf(
			"unsupported event type: %q",
			event.EventType,
		)
	}

	if event.OrderID <= 0 {
		return fmt.Errorf(
			"invalid order_id: %d",
			event.OrderID,
		)
	}

	return nil
}

type dbErrorKind string

const (
	dbRetryable    dbErrorKind = "retryable"
	dbNonRetryable dbErrorKind = "non_retryable"
	dbUnknown      dbErrorKind = "unknown"
)

func classifyPostgresError(err error) dbErrorKind {
	var pgErr *pgconn.PgError

	if !errors.As(err, &pgErr) {
		return dbUnknown
	}

	switch pgErr.Code {
	case "40001", "40P01":
		return dbRetryable

	case "22P02", "23503", "23514":
		return dbNonRetryable

	default:
		return dbUnknown
	}
}

func shouldRetryProcessing(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, errKafkaOffsetCommit) {
		return true
	}

	return classifyPostgresError(err) != dbNonRetryable
}

func retryDelay(attempt int) time.Duration {
	const maxDelay = 30 * time.Second

	delay := time.Second

	for i := 0; i < attempt && delay < maxDelay; i++ {
		delay *= 2

		if delay > maxDelay {
			delay = maxDelay
		}
	}

	jitter := time.Duration(
		rand.Int64N(int64(delay/5) + 1),
	)

	return delay - jitter
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println(".env not found, using environment variables")
	}

	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		broker = "localhost:9092"
	}

	client, err := kgo.NewClient(
		kgo.SeedBrokers(broker),
		kgo.ConsumeTopics("orders.events"),
		kgo.ConsumerGroup("delivery-idempotent-group"),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	dbCtx, dbCancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)

	db, err := database.New(dbCtx, database.Config{
		URL: os.Getenv("DATABASE_URL"),
	})
	dbCancel()

	if err != nil {
		log.Fatalf("PostgreSQL connection failed: %v", err)
	}
	defer db.Close()

	log.Println("PostgreSQL connected")

	shutdownCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	log.Println("Order Consumer started")

	for shutdownCtx.Err() == nil {
		fetches := client.PollRecords(shutdownCtx, 1)

		if shutdownCtx.Err() != nil {
			break
		}

		if errs := fetches.Errors(); len(errs) > 0 {
			log.Printf("Kafka fetch errors: %v", errs)
		}

		if fetches.NumRecords() == 0 {
			select {
			case <-shutdownCtx.Done():
			case <-time.After(time.Second):
			}
			continue
		}

		processingFailed := false

		fetches.EachRecord(func(record *kgo.Record) {
			attempt := 0

			var event OrderEvent

			err := json.Unmarshal(record.Value, &event)

			if err == nil {
				err = validateOrderEvent(event)
			}

			if err != nil {
				log.Printf("Invalid event payload: %v", err)
				handleErr := publishThenCommit(
					func() error {
						dlqCtx, cancel := context.WithTimeout(
							context.Background(),
							10*time.Second,
						)
						defer cancel()

						return publishToDLQ(dlqCtx, client, record, err)
					},
					func() error {
						commitCtx, cancel := context.WithTimeout(
							context.Background(),
							5*time.Second,
						)
						defer cancel()

						return client.CommitRecords(commitCtx, record)
					},
				)

				if handleErr != nil {
					log.Printf("DLQ handling failed: %v", handleErr)
					processingFailed = true
					return
				}

				log.Printf(
					"Invalid event sent to DLQ: partition=%d offset=%d",
					record.Partition,
					record.Offset,
				)
				return
			}

			for shutdownCtx.Err() == nil {
				processCtx, cancel := context.WithTimeout(
					context.Background(),
					30*time.Second,
				)

				isNew, err := processAndCommit(
					processCtx,
					db,
					event,
					func(commitCtx context.Context) error {
						return client.CommitRecords(commitCtx, record)
					},
				)
				cancel()

				if err == nil {
					if isNew {
						log.Printf("New event processed: %s", event.EventID)
					} else {
						log.Printf("Duplicate event skipped: %s", event.EventID)
					}

					log.Printf(
						"Kafka offset committed: partition=%d offset=%d",
						record.Partition,
						record.Offset,
					)
					return
				}

				if !shouldRetryProcessing(err) {
					log.Printf(
						"Non-retryable processing error: %v",
						err,
					)

					processingFailed = true
					return
				}

				delay := retryDelay(attempt)
				attempt++

				log.Printf(
					"Processing failed: %v. Attempt=%d, retrying in %s",
					err,
					attempt,
					delay,
				)

				select {
				case <-shutdownCtx.Done():
					return

				case <-time.After(delay):
				}

			}
		})

		if processingFailed {
			log.Println("Processing failed; stopping consumer to avoid skipping messages")
			break
		}
	}

	log.Printf("Shutdown reason: %v", context.Cause(shutdownCtx))
	log.Println("Order Consumer shutting down...")
}

func registerEvent(
	ctx context.Context,
	db *pgxpool.Pool,
	event OrderEvent,
) (bool, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	result, err := tx.Exec(ctx, `
		INSERT INTO processed_events (consumer_name, event_id)
		VALUES ($1, $2::uuid)
		ON CONFLICT (consumer_name, event_id) DO NOTHING
	`, "delivery-jobs-consumer-v1", event.EventID)

	if err != nil {
		return false, fmt.Errorf("register event: %w", err)
	}

	if result.RowsAffected() == 0 {
		return false, nil
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO delivery_jobs (order_id, event_id)
		VALUES ($1, $2::uuid)
	`, event.OrderID, event.EventID)

	if err != nil {
		return false, fmt.Errorf("create delivery job: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit transaction: %w", err)
	}

	return true, nil
}

func processAndCommit(
	ctx context.Context,
	db *pgxpool.Pool,
	event OrderEvent,
	commit func(context.Context) error,
) (bool, error) {
	isNew, err := registerEvent(ctx, db, event)
	if err != nil {
		return false, fmt.Errorf("process event: %w", err)
	}

	commitCtx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := commit(commitCtx); err != nil {
		return isNew, fmt.Errorf("%w: %w", errKafkaOffsetCommit, err)
	}

	return isNew, nil
}

func publishToDLQ(
	ctx context.Context,
	client *kgo.Client,
	record *kgo.Record,
	reason error,
) error {
	message := struct {
		SourceTopic     string `json:"source_topic"`
		SourcePartition int32  `json:"source_partition"`
		SourceOffset    int64  `json:"source_offset"`
		OriginalValue   []byte `json:"original_value"`
		Error           string `json:"error"`
	}{
		SourceTopic:     record.Topic,
		SourcePartition: record.Partition,
		SourceOffset:    record.Offset,
		OriginalValue:   record.Value,
		Error:           reason.Error(),
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal DLQ message: %w", err)
	}

	dlqRecord := &kgo.Record{
		Topic: "orders.events.dlq",
		Key:   record.Key,
		Value: payload,
	}

	if err := client.ProduceSync(ctx, dlqRecord).FirstErr(); err != nil {
		return fmt.Errorf("publish to DLQ: %w", err)
	}

	return nil
}

func publishThenCommit(
	publish func() error,
	commit func() error,
) error {
	if err := publish(); err != nil {
		return fmt.Errorf("publish to DLQ: %w", err)
	}

	if err := commit(); err != nil {
		return fmt.Errorf("commit Kafka offset: %w", err)
	}

	return nil
}
