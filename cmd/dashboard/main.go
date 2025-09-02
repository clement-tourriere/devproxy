package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"devproxy/internal/dashboard"
	"devproxy/internal/proxy"
)

func main() {
	// Handle health check flag
	if len(os.Args) > 1 && os.Args[1] == "--health" {
		fmt.Println("healthy")
		os.Exit(0)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create proxy manager to access container data
	manager, err := proxy.NewManager(logger)
	if err != nil {
		logger.Error("Failed to create proxy manager", "error", err)
		os.Exit(1)
	}

	// Create dashboard server
	server := dashboard.NewServer(manager, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Received shutdown signal")
		cancel()
	}()

	// Get listen address from environment or default
	addr := os.Getenv("DASHBOARD_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	logger.Info("Starting DevProxy Dashboard...")
	if err := server.Start(ctx, addr); err != nil {
		logger.Error("Dashboard server failed", "error", err)
		os.Exit(1)
	}

	logger.Info("DevProxy Dashboard stopped")
}
