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

	"github.com/ThDawnWind/food-delivery-api/internal/category"
	"github.com/ThDawnWind/food-delivery-api/internal/config"
	"github.com/ThDawnWind/food-delivery-api/internal/database"
	"github.com/ThDawnWind/food-delivery-api/internal/order"
	"github.com/ThDawnWind/food-delivery-api/internal/product"
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
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
	if err := godotenv.Load(); err != nil {
		log.Println(".env file not found, using environment variables")
	}

	serverErr := make(chan error, 1)
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}
	dbCtx, dbCancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer dbCancel()

	dbPool, err := database.New(dbCtx, cfg.Database.URL)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	} else {
		log.Println("Database connection established")
	}
	defer dbPool.Close()

	categoryRepository := category.NewRepository(dbPool)
	categoryService := category.NewService(categoryRepository)
	categoryHandler := category.NewHandler(categoryService)

	productRepository := product.NewRepository(dbPool)
	productService := product.NewService(productRepository)
	productHandler := product.NewHandler(productService)

	orderRepository := order.NewRepository(dbPool)
	orderService := order.NewService(
		productService,
		orderRepository,
	)
	orderHandler := order.NewHandler(orderService)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	router := chi.NewRouter()
	router.Get("/health", healthHandler)
	router.Get("/slow", slowHandler)

	router.Mount(
		"/api/v1/categories",
		categoryHandler.Routes(),
	)

	router.Mount(
		"/api/v1/products",
		productHandler.Routes(),
	)

	router.Mount(
		"/api/v1/orders",
		orderHandler.Routes(),
	)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler:           router,
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
