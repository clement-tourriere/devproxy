package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"devproxy/internal/config"
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

	manager, err := proxy.NewManager(cfg, logger)
	if err != nil {
		logger.Error("Failed to create manager", "error", err)
		os.Exit(1)
	}

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

	logger.Info("Starting DevProxy...")
	logger.Info("📋 Dashboard available at: https://devproxy-dashboard.localhost or http://devproxy-dashboard.localhost")
	logger.Info("💡 For HTTPS support: run './trust-cert.sh' then restart your browser")

	if err := manager.Start(ctx); err != nil {
		logger.Error("Manager failed", "error", err)
		os.Exit(1)
	}

	logger.Info("DevProxy stopped")
}
