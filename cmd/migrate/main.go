package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
)

func main() {
	err := run()
	if err != nil {
		log.Printf("migration failed: %v", err)
		os.Exit(1)
	}
}

func run() (runErr error) {
	err := godotenv.Load()
	if err != nil {
		log.Println(".env file not found, using environment variables")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("database URL is required")
	}

	if len(os.Args) < 2 {
		return errors.New("usage: migrate [up|down|status]")
	}

	command := os.Args[1]

	db, err := goose.OpenDBWithDriver(
		"postgres",
		databaseURL,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to open database: %w",
			err,
		)
	}

	defer func() {
		err := db.Close()
		if err != nil && runErr == nil {
			runErr = fmt.Errorf(
				"failed to close database: %w",
				err,
			)
		}
	}()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return fmt.Errorf(
			"failed to connect to database: %w",
			err,
		)
	}

	switch command {
	case "up":
		err = goose.Up(
			db,
			"migrations",
		)

	case "down":
		err = goose.Down(
			db,
			"migrations",
		)

	case "status":
		err = goose.Status(
			db,
			"migrations",
		)

	default:
		return fmt.Errorf(
			"unknown migration command %q",
			command,
		)
	}

	if err != nil {
		return fmt.Errorf(
			"migration command %q failed: %w",
			command,
			err,
		)
	}

	log.Print("migration completed successfully")

	return nil
}
