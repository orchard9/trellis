package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/orchard9/trellis/ingress/internal/auth"
	"github.com/orchard9/trellis/ingress/internal/ingestion"
	"github.com/orchard9/trellis/ingress/pkg/config"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Initialize Warden client for authentication
	wardenClient, err := auth.NewWardenClient(cfg.GetWardenAddress())
	if err != nil {
		slog.Error("failed to create warden client", "error", err)
		os.Exit(1)
	}
	defer wardenClient.Close()

	// Initialize ingestion components (placeholders for now)
	metrics := ingestion.NewSimpleMetrics()
	
	// TODO: Initialize actual pubsub, redis, clickhouse clients
	// For now, we'll use nil values and implement proper initialization later
	handler := ingestion.NewHandler(nil, nil, nil, metrics)

	// Setup HTTP router
	r := chi.NewRouter()

	// Middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(middleware.Compress(5))

	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // Configure appropriately for production
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Health checks (no authentication required)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Check dependencies (ClickHouse, Redis, Pub/Sub)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Ready"))
	})

	// Traffic ingestion routes (require authentication)
	r.Group(func(r chi.Router) {
		r.Use(wardenClient.AuthenticationMiddleware)

		// Main ingestion endpoints
		r.HandleFunc("/in", handler.HandleTraffic)
		r.HandleFunc("/in/{campaign_id}", handler.HandleTraffic)
		r.Get("/pixel.gif", handler.HandlePixel)
		r.HandleFunc("/postback", handler.HandlePostback)
	})

	// API routes (require authentication)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(wardenClient.AuthenticationMiddleware)

		// Health endpoint
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			orgCtx, ok := auth.GetOrganizationContext(r.Context())
			if !ok {
				http.Error(w, "Organization context not found", http.StatusInternalServerError)
				return
			}

			response := map[string]interface{}{
				"status":          "healthy",
				"service":         "trellis-ingress",
				"organization_id": orgCtx.OrganizationID,
				"timestamp":       time.Now().Unix(),
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"status": "%s", "service": "%s", "organization_id": "%s", "timestamp": %d}`,
				response["status"], response["service"], response["organization_id"], response["timestamp"])
		})

		// TODO: Add campaign management endpoints
		// r.Route("/campaigns", func(r chi.Router) {
		//     r.Get("/", listCampaigns)
		//     r.Post("/", createCampaign)
		//     r.Get("/{campaignID}", getCampaign)
		//     r.Put("/{campaignID}", updateCampaign)
		//     r.Delete("/{campaignID}", deleteCampaign)
		// })
	})

	// Setup HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		slog.Info("starting ingress server", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			cancel()
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case <-ctx.Done():
		slog.Info("context cancelled")
	case sig := <-sigChan:
		slog.Info("received signal, shutting down", "signal", sig)
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}

	slog.Info("ingress server stopped")
}