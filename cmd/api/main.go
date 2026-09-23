package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThDawnWind/food-delivery-api/internal/address"
	"github.com/ThDawnWind/food-delivery-api/internal/auth"
	"github.com/ThDawnWind/food-delivery-api/internal/category"
	"github.com/ThDawnWind/food-delivery-api/internal/config"
	"github.com/ThDawnWind/food-delivery-api/internal/database"
	"github.com/ThDawnWind/food-delivery-api/internal/order"
	"github.com/ThDawnWind/food-delivery-api/internal/product"
	"github.com/ThDawnWind/food-delivery-api/internal/user"
	"github.com/ThDawnWind/food-delivery-api/openapi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

type databasePinger interface {
	Ping(context.Context) error
}

func readinessHandler(db databasePinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			http.Error(
				w,
				"Service Unavailable",
				http.StatusServiceUnavailable,
			)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
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

	dbPool, err := database.New(
		dbCtx,
		database.Config{
			URL:               cfg.Database.URL,
			MaxConns:          cfg.Database.MaxConns,
			MinConns:          cfg.Database.MinConns,
			MaxConnLifetime:   cfg.Database.MaxConnLifetime,
			MaxConnIdleTime:   cfg.Database.MaxConnIdleTime,
			HealthCheckPeriod: cfg.Database.HealthCheckPeriod,
		},
	)
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

	addressRepository := address.NewRepository(dbPool)
	addressService := address.NewService(addressRepository)
	addressHandler := address.NewHandler(addressService)

	orderRepository := order.NewRepository(dbPool)
	orderService := order.NewService(
		productService,
		orderRepository,
		addressService,
	)
	orderHandler := order.NewHandlerWithLocation(
		orderService,
		location,
	)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Get("/health", healthHandler)
	router.Get("/ready", readinessHandler(dbPool))

	router.Get("/openapi.yaml", openapi.SpecHandler)
	router.Get("/docs", openapi.DocsHandler)

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
			r.Put("/{id}", productHandler.Update)
			r.Delete("/{id}", productHandler.Delete)
		})
	})

	router.Route("/api/v1/admin", func(r chi.Router) {
		r.Use(auth.Middleware(tokenManager))
		r.Use(auth.RequireRole("admin"))

		r.Get("/orders", orderHandler.ListAll)
		r.Get("/orders/{id}", orderHandler.AdminGetByID)
		r.Patch("/orders/{id}/status", orderHandler.UpdateStatus)
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
