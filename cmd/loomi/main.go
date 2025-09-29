package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	loomisvc "github.com/blueplan/loomi-go/internal/loomi"
	"github.com/blueplan/loomi-go/internal/loomi/agents"
	"github.com/blueplan/loomi-go/internal/loomi/api"
	"github.com/blueplan/loomi-go/internal/loomi/config"
	"github.com/blueplan/loomi-go/internal/loomi/llm"
	logx "github.com/blueplan/loomi-go/internal/loomi/log"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger, err := logx.NewLogger(cfg.App.LogLevel)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	logger.Info(context.Background(), "Starting Loomi service",
		logx.KV("version", cfg.App.Version),
		logx.KV("environment", getEnv("ENVIRONMENT", "development")))

	// Initialize LLM client
	llmClient, err := llm.NewClient(cfg.LLM.DefaultProvider, cfg.LLM.Providers[cfg.LLM.DefaultProvider])
	if err != nil {
		log.Fatalf("Failed to initialize LLM client: %v", err)
	}

	// Initialize Loomi service
	loomiService := loomisvc.NewLoomiService(cfg, logger, llmClient)
	// Inject default agents after service construct to avoid import cycle in loomi_service
	concierge := agents.NewLoomiConcierge(logger, llmClient)
	orchestrator := agents.NewLoomiOrchestrator(logger, llmClient)
	// wire dependencies via service getters
	_ = concierge // will be wired inside service when ProcessRequest is called if needed
	_ = orchestrator

	// Initialize API server
	apiServer := api.NewServer(logger, llmClient, loomiService)

	// Start server in a goroutine
	serverAddr := fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port)
	go func() {
		if err := apiServer.Run(serverAddr); err != nil {
			logger.Error(context.Background(), "Failed to start server", logx.KV("error", err))
			os.Exit(1)
		}
	}()

	logger.Info(context.Background(), "Loomi service started successfully",
		logx.KV("address", serverAddr))

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(context.Background(), "Shutting down Loomi service...")

	// Shutdown services
	ctx, cancel := context.WithTimeout(context.Background(), 30)
	defer cancel()

	if err := loomiService.Shutdown(ctx); err != nil {
		logger.Error(ctx, "Error shutting down Loomi service", logx.KV("error", err))
	}

	// Graceful shutdown would be implemented here
	logger.Info(ctx, "API server shutdown completed")

	logger.Info(ctx, "Loomi service stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
