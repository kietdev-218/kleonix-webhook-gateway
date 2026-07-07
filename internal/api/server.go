package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kleonix/webhook-gateway/internal/config"
	"github.com/kleonix/webhook-gateway/internal/handler"
	"github.com/kleonix/webhook-gateway/internal/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type Server struct {
	httpServer *http.Server
	logger     *zap.Logger
}

func NewServer(
	cfg *config.Config,
	kratosHandler *handler.KratosHandler,
	healthHandler *handler.Handler,
	logger *zap.Logger,
) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(middleware.CorrelationID())
	router.Use(middleware.Logging(logger))
	router.Use(middleware.Metrics())
	router.Use(middleware.SizeLimit(cfg.MaxRequestSize))
	router.Use(middleware.Timeout(cfg.RequestTimeout))

	// Observability
	router.GET("/health", healthHandler.Health)
	router.GET("/ready", healthHandler.Ready)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Webhooks
	v1 := router.Group("/webhooks")
	v1.Use(middleware.Auth(cfg.WebhookSecret))
	{
		v1.POST("/kratos", kratosHandler.HandleWebhook)
	}

	srv := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: router,
	}

	return &Server{
		httpServer: srv,
		logger:     logger,
	}
}

func (s *Server) Start() error {
	s.logger.Info("Starting HTTP server", zap.String("addr", s.httpServer.Addr))
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping HTTP server")
	return s.httpServer.Shutdown(ctx)
}
