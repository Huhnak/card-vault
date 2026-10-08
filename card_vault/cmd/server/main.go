package main

import (
	"context"
	"errors"
	"huhnak/card_vault/internal/application"
	httpinterface "huhnak/card_vault/internal/interface/http"
	"huhnak/card_vault/internal/repository/postgres"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	slog.Info("Start initializing server...")

	// Initializing postgres
	dbPool, err := pgxpool.New(context.Background(), os.Getenv("POSTGRES_URL"))
	if err != nil {
		slog.Error("Unable to connect to database: %v\n", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	db := stdlib.OpenDBFromPool(dbPool)
	defer db.Close()
	// End of initializing postgres

	repo := postgres.NewCardRepository(db)
	svc := application.NewCardService(&repo, nil)
	httpHandler := httpinterface.NewHandler(svc)

	mux := http.NewServeMux()
	httpHandler.RegisterRouter(mux)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("HTTP server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("received shutdown signal", "signal", sig.String())
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	} else {
		slog.Info("server gracefully stopped")
	}

}
