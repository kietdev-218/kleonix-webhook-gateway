package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kleonix/webhook-gateway/internal/api"
	"github.com/kleonix/webhook-gateway/internal/config"
	"github.com/kleonix/webhook-gateway/internal/handler"
	"github.com/kleonix/webhook-gateway/internal/logger"
	"github.com/kleonix/webhook-gateway/internal/rabbitmq"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	zLogger, err := logger.New(cfg.LogLevel)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer func() { _ = zLogger.Sync() }()

	rmqConn, err := rabbitmq.NewConnection(cfg.RabbitMQURL, zLogger)
	if err != nil {
		zLogger.Fatal("Failed to connect to RabbitMQ", zap.Error(err))
	}
	defer rmqConn.Close()

	if err := rmqConn.SetupExchange(cfg.RabbitMQExchange); err != nil {
		zLogger.Fatal("Failed to setup exchange", zap.Error(err))
	}

	pub := rabbitmq.NewRabbitMQPublisher(rmqConn, cfg.RabbitMQExchange, cfg.RabbitMQRoutingKey, cfg.RabbitMQMaxRetries, cfg.RabbitMQRetryDelay, zLogger)
	defer pub.Close()

	kratosHandler := handler.NewKratosHandler(pub, zLogger)
	healthHandler := handler.NewHandler()

	srv := api.NewServer(cfg, kratosHandler, healthHandler, zLogger)

	go func() {
		if err := srv.Start(); err != nil {
			zLogger.Fatal("Server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	zLogger.Info("Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Stop(ctx); err != nil {
		zLogger.Error("Server shutdown error", zap.Error(err))
	}

	zLogger.Info("Server exited")
}
