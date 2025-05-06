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
	"errors"
	"time"

	"github.com/google/uuid"
)

// ItemType defines the type of backlog item
type ItemType string

const (
	// ItemTypeEpic represents an epic
	ItemTypeEpic ItemType = "EPIC"
	// ItemTypeFeature represents a feature
	ItemTypeFeature ItemType = "FEATURE"
	// ItemTypeStory represents a user story
	ItemTypeStory ItemType = "STORY"
)

// ItemStatus defines the status of backlog item
type ItemStatus string

const (
	// ItemStatusNew represents a newly created item
	ItemStatusNew ItemStatus = "NEW"
	// ItemStatusReady represents an item ready for sprint
	ItemStatusReady ItemStatus = "READY"
	// ItemStatusInProgress represents an item in progress
	ItemStatusInProgress ItemStatus = "IN_PROGRESS"
	// ItemStatusDone represents a completed item
	ItemStatusDone ItemStatus = "DONE"
	// ItemStatusBlocked represents a blocked item
	ItemStatusBlocked ItemStatus = "BLOCKED"
)

// BacklogItem represents a backlog item (epic, feature, or story)
type BacklogItem struct {
	ID          uuid.UUID  `json:"id"`
	Type        ItemType   `json:"type"`
	ParentID    *uuid.UUID `json:"parentId"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	StoryPoints int        `json:"storyPoints"`
	Status      ItemStatus `json:"status"`
	Priority    int        `json:"priority"`
	Assignee    string     `json:"assignee"`
	Tags        []string   `json:"tags"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	ExternalIDs map[string]string `json:"externalIds"` // Map of external system IDs (e.g., "jira": "PROJ-123")
}

// NewBacklogItem creates a new backlog item
func NewBacklogItem(itemType ItemType, title, description string) (*BacklogItem, error) {
	if title == "" {
		return nil, errors.New("title cannot be empty")
	}

	if !isValidItemType(itemType) {
		return nil, errors.New("invalid item type")
	}

	now := time.Now().UTC()
	return &BacklogItem{
		ID:          uuid.New(),
		Type:        itemType,
		Title:       title,
		Description: description,
		Status:      ItemStatusNew,
		Priority:    0,
		Tags:        []string{},
		CreatedAt:   now,
		UpdatedAt:   now,
		ExternalIDs: make(map[string]string),
	}, nil
}

// UpdateTitle updates the item title
func (i *BacklogItem) UpdateTitle(title string) error {
	if title == "" {
		return errors.New("title cannot be empty")
	}
	i.Title = title
	i.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateDescription updates the item description
func (i *BacklogItem) UpdateDescription(description string) {
	i.Description = description
	i.UpdatedAt = time.Now().UTC()
}

// UpdateStatus updates the item status
func (i *BacklogItem) UpdateStatus(status ItemStatus) error {
	if !isValidItemStatus(status) {
		return errors.New("invalid item status")
	}
	i.Status = status
	i.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateStoryPoints updates story points
func (i *BacklogItem) UpdateStoryPoints(points int) error {
	if points < 0 {
		return errors.New("story points cannot be negative")
	}
	i.StoryPoints = points
	i.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdatePriority updates the item priority
func (i *BacklogItem) UpdatePriority(priority int) {
	i.Priority = priority
	i.UpdatedAt = time.Now().UTC()
}

// UpdateParent links the item to a parent
func (i *BacklogItem) UpdateParent(parentID *uuid.UUID) error {
	// Validate parent-child relationship based on item type
	if parentID != nil && i.Type == ItemTypeEpic {
		return errors.New("epic cannot have a parent")
	}
	i.ParentID = parentID
	i.UpdatedAt = time.Now().UTC()
	return nil
}

// AddTag adds a tag to the item
func (i *BacklogItem) AddTag(tag string) {
	for _, existingTag := range i.Tags {
		if existingTag == tag {
			return // Tag already exists
		}
	}
	i.Tags = append(i.Tags, tag)
	i.UpdatedAt = time.Now().UTC()
}

// RemoveTag removes a tag from the item
func (i *BacklogItem) RemoveTag(tag string) {
	for idx, existingTag := range i.Tags {
		if existingTag == tag {
			i.Tags = append(i.Tags[:idx], i.Tags[idx+1:]...)
			i.UpdatedAt = time.Now().UTC()
			return
		}
	}
}

// SetExternalID sets an external system ID
func (i *BacklogItem) SetExternalID(system, externalID string) {
	i.ExternalIDs[system] = externalID
	i.UpdatedAt = time.Now().UTC()
}

// GetExternalID retrieves an external system ID
func (i *BacklogItem) GetExternalID(system string) string {
	return i.ExternalIDs[system]
}

// IsReady checks if item is ready to be worked on
func (i *BacklogItem) IsReady() bool {
	return i.Status == ItemStatusReady
}

// Helper functions
func isValidItemType(itemType ItemType) bool {
	return itemType == ItemTypeEpic || itemType == ItemTypeFeature || itemType == ItemTypeStory
}

func isValidItemStatus(status ItemStatus) bool {
	return status == ItemStatusNew || 
		status == ItemStatusReady || 
		status == ItemStatusInProgress || 
		status == ItemStatusDone || 
		status == ItemStatusBlocked
}
