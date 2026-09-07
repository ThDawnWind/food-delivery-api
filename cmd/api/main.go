package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/ThDawnWind/food-delivery-api/internal/config"
	"github.com/ThDawnWind/food-delivery-api/internal/database"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("OK"))
	if err != nil {
		log.Printf("Error writing response: %v", err)
	} else {
		log.Println("Health check responded with OK")
	}
}

func slowHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for /slow, simulating slow response...")
	time.Sleep(4 * time.Second)
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("Slow response completed"))
	if err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func main() {
	serverErr := make(chan error, 1)
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbPool, err := database.New(ctx, cfg.Database.URL)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	} else {
		log.Println("Database connection established")
	}
	defer dbPool.Close()

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /slow", slowHandler)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler:           mux,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	go func() {
		log.Printf("Starting server on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("Received shutdown signal")
	case err := <-serverErr:
		log.Printf("Server error: %v", err)
		return
	}

	log.Println("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		6*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
		return
	}

	log.Println("Server gracefully stopped")
}
