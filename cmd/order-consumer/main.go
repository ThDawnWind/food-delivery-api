package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/ThDawnWind/food-delivery-api/internal/database"
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

func main() {
	client, err := kgo.NewClient(
		kgo.SeedBrokers("localhost:9092"),
		kgo.ConsumeTopics("orders.events"),
		kgo.ConsumerGroup("delivery-idempotent-group"),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	if err := godotenv.Load(); err != nil {
		log.Println(".env not found, using environment variables")
	}

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
			var event OrderEvent

			if err := json.Unmarshal(record.Value, &event); err != nil {
				log.Printf("Failed to decode event: %v", err)
				processingFailed = true
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

				log.Printf(
					"Processing failed: %v. Retrying in 3 seconds...",
					err,
				)

				select {
				case <-shutdownCtx.Done():
					return
				case <-time.After(3 * time.Second):

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
		return isNew, fmt.Errorf("commit Kafka offset: %w", err)
	}

	return isNew, nil
}
