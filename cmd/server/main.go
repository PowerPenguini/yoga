package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"main/di"
	"main/handlers"
)

func main() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres@localhost:5432/yoga?sslmode=disable"
	}

	deps, err := di.NewDI(connStr)
	if err != nil {
		log.Fatalf("initialize DI: %v", err)
	}
	defer deps.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()
	healthHandler := handlers.NewHealthHandler(deps)
	itemHandler := handlers.NewItemsHandler(deps)

	mux.HandleFunc("GET /health", healthHandler.GetHealth)
	mux.HandleFunc("POST /items", itemHandler.PostItem)
	mux.HandleFunc("GET /items", itemHandler.GetItems)
	mux.HandleFunc("PATCH /items/{item_id}", itemHandler.PatchItem)
	mux.HandleFunc("DELETE /items/{item_id}", itemHandler.DeleteItem)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	serverErr := make(chan error, 1)
	go func() {
		deps.Logger.Println("server running on :8080")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case <-ctx.Done():
		deps.Logger.Println("shutdown signal received")
	case err := <-serverErr:
		deps.Logger.Fatalf("http server error: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		deps.Logger.Printf("http shutdown error: %v", err)
	}
}
