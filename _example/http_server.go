//go:build run

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/OutOfBedlam/tailer"
)

func main() {
	handler := tailer.Handler(
		"/",               // http path prefix to cut
		"/var/log/syslog", // file to tail
		tailer.WithPlugins(tailer.NewColoring("molokai")), // options
	)

	server := &http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: handler,
	}

	// Start server in goroutine
	go func() {
		log.Println("Server starting on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Signal all SSE connections to close
	tailer.Shutdown()

	// Gracefully shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
