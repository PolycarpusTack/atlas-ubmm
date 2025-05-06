// services/backlog-service/cmd/main.go

package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/ubmm/backlog-service/internal/adapters/db"
	"github.com/ubmm/backlog-service/internal/adapters/eventbus"
	"github.com/ubmm/backlog-service/internal/adapters/cache"
	"github.com/ubmm/backlog-service/internal/adapters/grpc"
	"github.com/ubmm/backlog-service/internal/config"
	"github.com/ubmm/backlog-service/internal/domain/service"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Errorf("failed to initialize logger: %w", err))
	}
	defer logger.Sync()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Initialize database
	dbAdapter, err := db.NewPostgresAdapter(cfg.Database)
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}
	defer dbAdapter.Close()

	// Initialize cache
	cacheAdapter, err := cache.NewRedisAdapter(cfg.Cache)
	if err != nil {
		logger.Fatal("Failed to initialize cache", zap.Error(err))
	}
	defer cacheAdapter.Close()

	// Initialize event bus
	eventBusAdapter, err := eventbus.NewKafkaAdapter(cfg.EventBus)
	if err != nil {
		logger.Fatal("Failed to initialize event bus", zap.Error(err))
	}
	defer eventBusAdapter.Close()

	// Initialize domain service
	domainService := service.NewBacklogService(dbAdapter, cacheAdapter, eventBusAdapter)

	// Create gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpc.UnaryServerInterceptor()),
		grpc.StreamInterceptor(grpc.StreamServerInterceptor()),
	)

	// Register gRPC services
	backlogServer := grpc.NewBacklogServer(domainService, logger)
	pb.RegisterBacklogServiceServer(grpcServer, backlogServer)

	// Register health check
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// Enable reflection
	reflection.Register(grpcServer)

	// Start gRPC server
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPCPort))
	if err != nil {
		logger.Fatal("Failed to listen", zap.Error(err))
	}

	go func() {
		logger.Info("Starting gRPC server", zap.Int("port", cfg.Server.GRPCPort))
		if err := grpcServer.Serve(listener); err != nil {
			logger.Fatal("Failed to serve gRPC", zap.Error(err))
		}
	}()

	// Start HTTP server for metrics and health
	httpMux := http.NewServeMux()
	httpMux.Handle("/metrics", promhttp.Handler())
	httpMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.HTTPPort),
		Handler: httpMux,
	}

	go func() {
		logger.Info("Starting HTTP server", zap.Int("port", cfg.Server.HTTPPort))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to serve HTTP", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down servers...")

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("Failed to shutdown HTTP server", zap.Error(err))
	}

	// Shutdown gRPC server
	grpcServer.GracefulStop()

	logger.Info("Servers shutdown complete")
}

// services/backlog-service/internal/domain/model/item.go

package model

import (
	"time"

	"