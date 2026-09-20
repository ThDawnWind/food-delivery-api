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

	"github.com/ThDawnWind/food-delivery-api/internal/address"
	"github.com/ThDawnWind/food-delivery-api/internal/auth"
	"github.com/ThDawnWind/food-delivery-api/internal/category"
	"github.com/ThDawnWind/food-delivery-api/internal/config"
	"github.com/ThDawnWind/food-delivery-api/internal/database"
	"github.com/ThDawnWind/food-delivery-api/internal/order"
	"github.com/ThDawnWind/food-delivery-api/internal/product"
	"github.com/ThDawnWind/food-delivery-api/internal/user"
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

	location, err := time.LoadLocation(
		cfg.Timezone,
	)
	if err != nil {
		log.Fatalf(
			"failed to load timezone: %v",
			err,
		)
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

	userRepository := user.NewRepository(dbPool)
	userService := user.NewService(userRepository)

	tokenManager, err := auth.NewTokenManager(
		cfg.JWT.Secret,
		cfg.JWT.TTL,
	)
	if err != nil {
		log.Fatalf(
			"Error creating token manager: %v",
			err,
		)
	}

	authService := auth.NewService(
		userService,
		tokenManager,
	)

	authHandler := auth.NewHandler(authService)

	orderRepository := order.NewRepository(dbPool)
	orderService := order.NewService(
		productService,
		orderRepository,
	)
	orderHandler := order.NewHandlerWithLocation(
		orderService,
		location,
	)

	addressRepository := address.NewRepository(dbPool)
	addressService := address.NewService(addressRepository)
	addressHandler := address.NewHandler(addressService)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	router := chi.NewRouter()
	router.Get("/health", healthHandler)
	router.Get("/slow", slowHandler)

	router.Route("/api/v1/categories", func(r chi.Router) {
		r.Get("/", categoryHandler.List)
		r.Get("/{id}", categoryHandler.GetByID)

		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(tokenManager))
			r.Use(auth.RequireRole("admin"))

			r.Post("/", categoryHandler.Create)
			r.Put("/{id}", categoryHandler.Update)
			r.Delete("/{id}", categoryHandler.Delete)
		})
	})

	router.Route("/api/v1/products", func(r chi.Router) {
		r.Get("/", productHandler.List)
		r.Get("/{id}", productHandler.GetByID)

		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(tokenManager))
			r.Use(auth.RequireRole("admin"))

			r.Post("/", productHandler.Create)
			r.Patch("/{id}", productHandler.Update)
			r.Delete("/{id}", productHandler.Delete)
		})
	})

	router.Route("/api/v1/admin", func(r chi.Router) {
		r.Use(auth.Middleware(tokenManager))
		r.Use(auth.RequireRole("admin"))

		r.Get(
			"/orders",
			orderHandler.ListAll,
		)
		r.Get(
			"/orders/{id}",
			orderHandler.AdminGetByID,
		)
	})

	router.Group(func(r chi.Router) {
		r.Use(auth.Middleware(tokenManager))

		r.Mount(
			"/api/v1/orders",
			orderHandler.Routes(),
		)

		r.Mount(
			"/api/v1/addresses",
			addressHandler.Routes(),
		)
	})

	router.Mount(
		"/api/v1/auth",
		authHandler.Routes(),
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
