// Command server starts the math operations HTTP API on port 8000.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/betodalas/teste-sre-roberto/internal/api"
)

const (
	addr            = ":8000"
	shutdownTimeout = 10 * time.Second
)

func main() {
	server := &http.Server{
		Addr:         addr,
		Handler:      api.NewRouter(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	// Listen for SIGTERM/SIGINT so Kubernetes rolling updates can drain
	// in-flight requests instead of killing the process abruptly.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("starting server on %s", addr)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	case <-ctx.Done():
		log.Printf("shutdown signal received, draining connections (timeout %s)", shutdownTimeout)
		stop()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
			os.Exit(1)
		}
		log.Println("server stopped gracefully")
	}
}
