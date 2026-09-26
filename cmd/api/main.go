package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	"github.com/ThDawnWind/food-delivery-api/internal/httpx"
	"github.com/ThDawnWind/food-delivery-api/internal/order"
	"github.com/ThDawnWind/food-delivery-api/internal/product"
	"github.com/ThDawnWind/food-delivery-api/internal/user"
	"github.com/ThDawnWind/food-delivery-api/openapi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/time/rate"
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
	logger := slog.New(
		slog.NewJSONHandler(
			os.Stdout,
			&slog.HandlerOptions{
				Level: slog.LevelInfo,
			},
		),
	)

	if err := run(logger); err != nil {
		logger.Error(
			"application stopped with error",
			slog.Any("error", err),
		)

		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	registry := prometheus.NewRegistry()

	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(
			collectors.ProcessCollectorOpts{},
		),
	)

	httpMetrics := httpx.NewHTTPMetrics(
		registry,
	)

	if err := godotenv.Load(); err != nil {
		logger.Info(
			".env file not found, using environment variables",
		)
	}

	serverErr := make(chan error, 1)
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf(
			"failed to load configuration: %w",
			err,
		)
	}

	location, err := time.LoadLocation(
		cfg.Timezone,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to load timezone: %w",
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
		return fmt.Errorf(
			"failed to connect to database: %w",
			err,
		)
	}

	logger.Info(
		"database connection established",
	)

	dbPoolMetrics := database.NewPoolMetrics(
		dbPool,
	)

	registry.MustRegister(
		dbPoolMetrics,
	)
	defer dbPool.Close()

	categoryRepository := category.NewRepository(dbPool)
	categoryService := category.NewService(categoryRepository)
	categoryHandler := category.NewHandler(
		categoryService,
		logger,
	)

	productRepository := product.NewRepository(dbPool)
	productService := product.NewService(productRepository)
	productHandler := product.NewHandler(
		productService,
		logger,
	)

	userRepository := user.NewRepository(dbPool)
	userService := user.NewService(userRepository)

	tokenManager, err := auth.NewTokenManager(
		cfg.JWT.Secret,
		cfg.JWT.TTL,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to create token manager: %w",
			err,
		)
	}

	authService := auth.NewService(
		userService,
		tokenManager,
	)

	authHandler := auth.NewHandler(
		authService,
		logger,
	)

	authRateLimiter := httpx.NewIPRateLimiter(
		rate.Every(12*time.Second),
		5,
		15*time.Minute,
		cfg.HTTP.TrustedProxyCIDRs,
	)

	addressRepository := address.NewRepository(dbPool)
	addressService := address.NewService(addressRepository)
	addressHandler := address.NewHandler(
		addressService,
		logger,
	)

	orderRepository := order.NewRepository(dbPool)
	orderService := order.NewService(
		productService,
		orderRepository,
		addressService,
	)
	orderHandler := order.NewHandlerWithLocation(
		orderService,
		location,
		logger,
	)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(
		httpx.RequestLogger(
			logger,
			cfg.HTTP.TrustedProxyCIDRs,
		),
	)
	router.Use(
		httpx.MetricsMiddleware(
			httpMetrics,
		),
	)
	router.Use(
		httpx.Recoverer(
			logger,
		),
	)

	router.Use(httpx.SecurityHeaders)

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins: cfg.HTTP.CORSAllowedOrigins,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
		},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	router.Get("/health", healthHandler)
	router.Get("/ready", readinessHandler(dbPool))

	router.Handle(
		"/metrics",
		promhttp.HandlerFor(
			registry,
			promhttp.HandlerOpts{},
		),
	)

	router.Get("/openapi.yaml", openapi.SpecHandler)
	router.Get("/docs", openapi.DocsHandler)

	router.Route("/api/v1/categories", func(r chi.Router) {
		r.Get("/", categoryHandler.List)
		r.Get("/{id}", categoryHandler.GetByID)

		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(tokenManager))
			r.Use(
				auth.RequireRole(
					userService,
					user.RoleAdmin,
				),
			)

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
			r.Use(
				auth.RequireRole(
					userService,
					user.RoleAdmin,
				),
			)

			r.Post("/", productHandler.Create)
			r.Put("/{id}", productHandler.Update)
			r.Delete("/{id}", productHandler.Delete)
		})
	})

	router.Route("/api/v1/admin", func(r chi.Router) {
		r.Use(auth.Middleware(tokenManager))
		r.Use(
			auth.RequireRole(
				userService,
				user.RoleAdmin,
			),
		)

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

	router.Group(func(r chi.Router) {
		r.Use(authRateLimiter.Middleware)

		r.Mount(
			"/api/v1/auth",
			authHandler.Routes(),
		)
	})

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler:           router,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	go func() {
		logger.Info(
			"starting server",
			slog.String(
				"address",
				server.Addr,
			),
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info(
			"received shutdown signal",
		)
	case err := <-serverErr:
		return fmt.Errorf(
			"server error: %w",
			err,
		)
	}

	logger.Info(
		"shutting down server...",
	)

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		6*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf(
			"failed to shutdown server: %w",
			err,
		)
	}

	logger.Info(
		"server gracefully stopped",
	)

	return nil
}
