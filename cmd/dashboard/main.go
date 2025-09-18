package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"devproxy/internal/config"
	"devproxy/internal/dashboard"
	"devproxy/internal/proxy"
)

func main() {
	// Handle health check flag
	if len(os.Args) > 1 && os.Args[1] == "--health" {
		fmt.Println("healthy")
		os.Exit(0)
	}

	// Load configuration
	cfg := config.Load()

	// Set log level based on configuration
	var logLevel slog.Level
	switch cfg.DevProxy.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// Create proxy manager to access container data
	manager, err := proxy.NewManager(cfg, logger)
	if err != nil {
		logger.Error("Failed to create proxy manager", "error", err)
		os.Exit(1)
	}

	// Create dashboard server
	server := dashboard.NewServer(cfg, manager, logger)

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

	logger.Info("Starting DevProxy Dashboard...")
	if err := server.Start(ctx, cfg.Dashboard.Addr); err != nil {
		logger.Error("Dashboard server failed", "error", err)
		os.Exit(1)
	}

	logger.Info("DevProxy Dashboard stopped")
}
