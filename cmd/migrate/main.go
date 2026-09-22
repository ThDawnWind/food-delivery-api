package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println(".env file not found, using environment variables")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	if len(os.Args) < 2 {
		log.Fatal("usage: migrate [up|down|status]")
	}

	command := os.Args[1]

	db, err := goose.OpenDBWithDriver("postgres", databaseURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	switch command {
	case "up":
		err = goose.Up(db, "migrations")

	case "down":
		err = goose.Down(db, "migrations")

	case "status":
		err = goose.Status(db, "migrations")

	default:
		log.Fatalf("unknown command %q", command)
	}

	if err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	fmt.Printf("migration command %q completed successfully\n", command)
}
