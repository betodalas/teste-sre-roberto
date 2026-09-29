// Command server starts the math operations HTTP API on port 8000.
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/betodalas/teste-sre-roberto/internal/api"
)

const addr = ":8000"

func main() {
	server := &http.Server{
		Addr:         addr,
		Handler:      api.NewRouter(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	log.Printf("starting server on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
