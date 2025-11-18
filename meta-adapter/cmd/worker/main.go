package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/config"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/repository"
	"github.com/tarcisoamorim/whatsmeow/meta-adapter/internal/webhook"
	pkglogger "github.com/tarcisoamorim/whatsmeow/meta-adapter/pkg/logger"
)

const banner = `
╦ ╦┌─┐┌┐ ┬ ┬┌─┐┌─┐┬┌─  ╦ ╦┌─┐┬─┐┬┌─┌─┐┬─┐
║║║├┤ ├┴┐├─┤│ ││ │├┴┐  ║║║│ │├┬┘├┴┐├┤ ├┬┘
╚╩╝└─┘└─┘┴ ┴└─┘└─┘┴ ┴  ╚╩╝└─┘┴└─┴ ┴└─┘┴└─
   🔄 Async Webhook Delivery Worker
`

func main() {
	fmt.Println(banner)

	// Initialize logger
	if err := pkglogger.Init(); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer pkglogger.Get().Sync()

	zlog := pkglogger.Get()
	zlog.Info("Starting Webhook Worker")

	// Load config
	cfg, err := config.Load()
	if err != nil {
		zlog.Fatal("Failed to load config", zap.Error(err))
	}

	// Connect to database
	var db *repository.Database
	if cfg.Database.URL != "" {
		db, err = repository.NewDatabase(cfg.Database.URL)
		if err != nil {
			zlog.Fatal("Failed to connect to database", zap.Error(err))
		}
		defer db.Close()
		zlog.Info("Database connected successfully")
	} else {
		zlog.Fatal("DATABASE_URL not set, worker cannot run without database")
	}

	// Initialize webhook deliverer
	if cfg.RabbitMQ.URL == "" {
		zlog.Fatal("RABBITMQ_URL not set, worker cannot run without RabbitMQ")
	}

	deliverer, err := webhook.NewDeliverer(cfg.RabbitMQ.URL, cfg.Webhook.Secret, db)
	if err != nil {
		zlog.Fatal("Failed to initialize webhook deliverer", zap.Error(err))
	}
	defer deliverer.Close()

	zlog.Info("Webhook deliverer initialized")

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start deliverer in goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := deliverer.Start(ctx); err != nil {
			errCh <- err
		}
	}()

	zlog.Info("Worker started, waiting for webhook events...")

	// Wait for shutdown signal or error
	select {
	case <-sigCh:
		zlog.Info("Shutdown signal received, stopping worker...")
		cancel()
	case err := <-errCh:
		zlog.Error("Worker error", zap.Error(err))
		cancel()
	}

	zlog.Info("Webhook worker stopped")
}
