// services/backlog-service/internal/domain/service/backlog_service.go

package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/ubmm/backlog-service/internal/domain/model"
	"github.com/ubmm/backlog-service/internal/domain/repository"
	"github.com/ubmm/backlog-service/internal/domain/event"
)

// BacklogService implements the core business logic for backlog management
type BacklogService struct {
	repo          repository.BacklogRepository
	eventRepo     repository.EventRepository
	metricsRepo   repository.MetricsRepository
	eventPublisher event.Publisher
	cache         CacheProvider
	logger        *zap.Logger
}

// CacheProvider defines the interface for caching
type CacheProvider interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
}

// NewBacklogService creates a new instance of BacklogService
func NewBacklogService(
	repo repository.BacklogRepository,
	eventRepo repository.EventRepository,
	metricsRepo repository.MetricsRepository,
	eventPublisher event.Publisher,
	cache CacheProvider,
	logger *zap.Logger,
) *BacklogService {
	return &BacklogService{
		repo:          repo,
		eventRepo:     eventRepo,
		metricsRepo:   metricsRepo,
		eventPublisher: eventPublisher,
		cache:         cache,
		logger:        logger,
	}
}

// CreateItem creates a new backlog item
func (s *BacklogService) CreateItem(ctx context.Context, req *CreateItemRequest) (*model.BacklogItem, error) {
	// Create the backlog item
	item, err := model.NewBacklogItem(req.Type, req.Title, req.Description)
	if err != nil {
		return nil, err
	}

	// Set additional properties
	if req.ParentID != nil {
		err = item.UpdateParent(req.ParentID)
		if err != nil {
			return nil, err
		}

		// Validate parent exists and check parent-child relationship
		if req.ParentID != nil {
			parent, err := s.repo.GetByID(ctx, *req.ParentID)
			if err != nil {
				return nil, err
			}

			// Validate parent-child relationship
			if !isValidParentChild(parent.Type, req.Type) {
				return nil, errors.New("invalid parent-child relationship")
			}
		}
	}

	if req.StoryPoints > 0 {
		err = item.UpdateStoryPoints(req.StoryPoints)
		if err != nil {
			return nil, err
		}
	}

	// Add tags
	for _, tag := range req.Tags {
		item.AddTag(tag)
	}

	// Persist the item
	err = s.repo.Create(ctx, item)
	if err != nil {
		return nil, err
	}

	// Store event
	createEvent := event.NewItemCreatedEvent(item.ID, item)
	err = s.eventRepo.StoreEvent(ctx, createEvent)
	if err != nil {
		s.logger.Error("Failed to store item created event", zap.Error(err))
	}

	// Publish event
	err = s.eventPublisher.Publish(ctx, "backlog.item.created", createEvent)
	if err != nil {
		s.logger.Error("Failed to publish item created event", zap.Error(err))
	}

	// Invalidate cache
	s.invalidateListCache(ctx)

	return item, nil
}

// GetItem retrieves a backlog item by ID
func (s *BacklogService) GetItem(ctx context.Context, id uuid.UUID) (*model.BacklogItem, error) {
	// Try to get from cache first
	cacheKey := "item:" + id.String()
	cachedItem, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cachedItem != nil {
		if item, ok := cachedItem.(*model.BacklogItem); ok {
			return item, nil
		}
	}

	// Get from repository
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cache the result
	err = s.cache.Set(ctx, cacheKey, item, 1*time.Hour)
	if err != nil {
		s.logger.Error("Failed to cache item", zap.Error(err))
	}

	return item, nil
}

// UpdateItem updates an existing backlog item
func (s *BacklogService) UpdateItem(ctx context.Context, id uuid.UUID, req *UpdateItemRequest) (*model.BacklogItem, error) {
	// Get the existing item
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Title != nil {
		err = item.UpdateTitle(*req.Title)
		if err != nil {
			return nil, err
		}
	}

	if req.Description != nil {
		item.UpdateDescription(*req.Description)
	}

	if req.Status != nil {
		err = item.UpdateStatus(*req.Status)
		if err != nil {
			return nil, err
		}
	}

	if req.StoryPoints != nil {
		err = item.UpdateStoryPoints(*req.StoryPoints)
		if err != nil {
			return nil, err
		}
	}

	if req.Priority != nil {
		item.UpdatePriority(*req.Priority)
	}

	if req.ParentID != nil {
		if *req.ParentID != uuid.Nil {
			// Validate parent exists and check parent-child relationship
			parent, err := s.repo.GetByID(ctx, *req.ParentID)
			if err != nil {
				return nil, err
			}

			// Validate parent-child relationship
			if !isValidParentChild(parent.Type, item.Type) {
				return nil, errors.New("invalid parent-child relationship")
			}
		}

		err = item.UpdateParent(req.ParentID)
		if err != nil {
			return nil, err
		}
	}

	if req.Assignee != nil {
		item.Assignee = *req.Assignee
	}

	// Update tags if provided
	if req.Tags != nil {
		// Clear existing tags and add new ones
		item.Tags = []string{}
		for _, tag := range *req.Tags {
			item.AddTag(tag)
		}
	}

	// Persist the updated item
	err = s.repo.Update(ctx, item)
	if err != nil {
		return nil, err
	}

	// Store event
	updateEvent := event.NewItemUpdatedEvent(item.ID, item)
	err = s.eventRepo.StoreEvent(ctx, updateEvent)
	if err != nil {
		s.logger.Error("Failed to store item updated event", zap.Error(err))
	}

	// Publish event
	err = s.eventPublisher.Publish(ctx, "backlog.item.updated", updateEvent)
	if err != nil {
		s.logger.Error("Failed to publish item updated event", zap.Error(err))
	}

	// Invalidate caches
	s.cache.Delete(ctx, "item:"+id.String())
	s.invalidateListCache(ctx)

	return item, nil
}

// DeleteItem deletes a backlog item
func (s *BacklogService) DeleteItem(ctx context.Context, id uuid.UUID) error {
	// Check if item exists
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Check if item has children
	children, err := s.repo.GetChildren(ctx, id)
	if err != nil {
		return err
	}

	if len(children) > 0 {
		return errors.New("cannot delete item with children")
	}

	// Delete the item
	err = s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}

	// Store event
	deleteEvent := event.NewItemDeletedEvent(item.ID, item)
	err = s.eventRepo.StoreEvent(ctx, deleteEvent)
	if err != nil {
		s.logger.Error("Failed to store item deleted event", zap.Error(err))
	}

	// Publish event
	err = s.eventPublisher.Publish(ctx, "backlog.item.deleted", deleteEvent)
	if err != nil {
		s.logger.Error("Failed to publish item deleted event", zap.Error(err))
	}

	// Invalidate caches
	s.cache.Delete(ctx, "item:"+id.String())
	s.invalidateListCache(ctx)

	return nil
}

// ListItems lists backlog items with filtering
func (s *BacklogService) ListItems(ctx context.Context, filter repository.BacklogFilter) ([]*model.BacklogItem, int64, error) {
	// Try to get from cache if no search query
	if filter.SearchQuery == "" {
		cacheKey := buildListCacheKey(filter)
		cachedResult, err := s.cache.Get(ctx, cacheKey)
		if err == nil && cachedResult != nil {
			if result, ok := cachedResult.(*listCacheResult); ok {
				return result.Items, result.TotalCount, nil
			}
		}
	}

	// Get from repository
	items, totalCount, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Cache the result if no search query
	if filter.SearchQuery == "" {
		cacheKey := buildListCacheKey(filter)
		cacheResult := &listCacheResult{
			Items:      items,
			TotalCount: totalCount,
		}
		err = s.cache.Set(ctx, cacheKey, cacheResult, 5*time.Minute)
		if err != nil {
			s.logger.Error("Failed to cache list result", zap.Error(err))
		}
	}

	return items, totalCount, nil
}

// GetChildren retrieves all children of a backlog item
func (s *BacklogService) GetChildren(ctx context.Context, parentID uuid.UUID) ([]*model.BacklogItem, error) {
	// Try to get from cache
	cacheKey := "children:" + parentID.String()
	cachedResult, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cachedResult != nil {
		if children, ok := cachedResult.([]*model.BacklogItem); ok {
			return children, nil
		}
	}

	// Get from repository
	children, err := s.repo.GetChildren(ctx, parentID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	err = s.cache.Set(ctx, cacheKey, children, 5*time.Minute)
	if err != nil {
		s.logger.Error("Failed to cache children", zap.Error(err))
	}

	return children, nil
}

// ReorderItems reorders backlog items by updating their priorities
func (s *BacklogService) ReorderItems(ctx context.Context, reorderRequests []ReorderRequest) error {
	if len(reorderRequests) == 0 {
		return nil
	}

	// Create a map of item IDs to new priorities
	itemPriorities := make(map[uuid.UUID]int)
	for _, req := range reorderRequests {
		itemPriorities[req.ItemID] = req.NewPriority
	}

	// Update priorities in a batch
	err := s.repo.UpdatePriorities(ctx, itemPriorities)
	if err != nil {
		return err
	}

	// Store event
	reorderEvent := event.NewItemsReorderedEvent(itemPriorities)
	err = s.eventRepo.StoreEvent(ctx, reorderEvent)
	if err != nil {
		s.logger.Error("Failed to store items reordered event", zap.Error(err))
	}

	// Publish event
	err = s.eventPublisher.Publish(ctx, "backlog.items.reordered", reorderEvent)
	if err != nil {
		s.logger.Error("Failed to publish items reordered event", zap.Error(err))
	}

	// Invalidate list caches
	s.invalidateListCache(ctx)

	return nil
}

// SetExternalID sets an external system ID for a backlog item
func (s *BacklogService) SetExternalID(ctx context.Context, id uuid.UUID, system, externalID string) error {
	// Get the existing item
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Set the external ID
	item.SetExternalID(system, externalID)

	// Persist the updated item
	err = s.repo.Update(ctx, item)
	if err != nil {
		return err
	}

	// Store event
	externalIDEvent := event.NewExternalIDSetEvent(item.ID, system, externalID)
	err = s.eventRepo.StoreEvent(ctx, externalIDEvent)
	if err != nil {
		s.logger.Error("Failed to store external ID event", zap.Error(err))
	}

	// Publish event
	err = s.eventPublisher.Publish(ctx, "backlog.item.external_id.set", externalIDEvent)
	if err != nil {
		s.logger.Error("Failed to publish external ID event", zap.Error(err))
	}

	// Invalidate item cache
	s.cache.Delete(ctx, "item:"+id.String())

	return nil
}

// GetMetrics retrieves backlog metrics
func (s *BacklogService) GetMetrics(ctx context.Context) (*BacklogMetrics, error) {
	// Try to get from cache
	cacheKey := "metrics"
	cachedResult, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cachedResult != nil {
		if metrics, ok := cachedResult.(*BacklogMetrics); ok {
			return metrics, nil
		}
	}

	// Get backlog size
	sizeCounts, err := s.metricsRepo.GetBacklogSize(ctx)
	if err != nil {
		return nil, err
	}

	// Get average item age
	ageMetrics, err := s.metricsRepo.GetItemAge(ctx, model.ItemStatusNew)
	if err != nil {
		return nil, err
	}

	// Get WIP count
	wipCount, err := s.metricsRepo.GetWIPCounts(ctx)
	if err != nil {
		return nil, err
	}

	// Get lead time
	leadTime, err := s.metricsRepo.GetLeadTime(ctx, 30)
	if err != nil {
		return nil, err
	}

	// Get throughput
	throughput, err := s.metricsRepo.GetThroughput(ctx, 30)
	if err != nil {
		return nil, err
	}

	// Build metrics response
	metrics := &BacklogMetrics{
		TotalItems:    sizeCounts[model.ItemTypeEpic] + sizeCounts[model.ItemTypeFeature] + sizeCounts[model.ItemTypeStory],
		EpicCount:     sizeCounts[model.ItemTypeEpic],
		FeatureCount:  sizeCounts[model.ItemTypeFeature],
		StoryCount:    sizeCounts[model.ItemTypeStory],
		AverageAge:    calculateAverageAge(ageMetrics),
		WIPCount:      wipCount,
		LeadTimeDays:  leadTime,
		ThroughputLast30Days: throughput,
		IcebergRatio:  calculateIcebergRatio(sizeCounts),
		HealthStatus:  determineHealthStatus(sizeCounts, wipCount, leadTime),
	}

	// Cache the result
	err = s.cache.Set(ctx, cacheKey, metrics, 1*time.Hour)
	if err != nil {
		s.logger.Error("Failed to cache metrics", zap.Error(err))
	}

	return metrics, nil
}

// Helper functions

func isValidParentChild(parentType, childType model.ItemType) bool {
	if parentType == model.ItemTypeEpic && childType == model.ItemTypeFeature {
		return true
	}
	if parentType == model.ItemTypeFeature && childType == model.ItemTypeStory {
		return true
	}
	return false
}

func (s *BacklogService) invalidateListCache(ctx context.Context) {
	// This would be more sophisticated in a real system
	// For simplicity, we're just invalidating a few fixed cache keys
	s.cache.Delete(ctx, "list:all")
	s.cache.Delete(ctx, "metrics")
}

func buildListCacheKey(filter repository.BacklogFilter) string {
	// A real implementation would build a more sophisticated cache key based on all filter parameters
	return "list:all"
}

func calculateAverageAge(ageMetrics map[model.ItemType]float64) float64 {
	total := 0.0
	count := 0

	for _, age := range ageMetrics {
		total += age
		count++
	}

	if count == 0 {
		return 0
	}

	return total / float64(count)
}

func calculateIcebergRatio(sizeCounts map[model.ItemType]int) float64 {
	total := sizeCounts[model.ItemTypeEpic] + sizeCounts[model.ItemTypeFeature] + sizeCounts[model.ItemTypeStory]
	if total == 0 {
		return 0
	}

	epicRatio := float64(sizeCounts[model.ItemTypeEpic]) / float64(total)
	featureRatio := float64(sizeCounts[model.ItemTypeFeature]) / float64(total)
	storyRatio := float64(sizeCounts[model.ItemTypeStory]) / float64(total)

	// Ideal iceberg ratio is 1/3 for each type
	deviation := abs(epicRatio-0.33) + abs(featureRatio-0.33) + abs(storyRatio-0.33)
	
	// Convert to a score between 0 and 1 where 1 is perfect
	return 1.0 - (deviation / 2.0)
}

func determineHealthStatus(sizeCounts map[model.ItemType]int, wipCount int, leadTime float64) string {
	totalItems := sizeCounts[model.ItemTypeEpic] + sizeCounts[model.ItemTypeFeature] + sizeCounts[model.ItemTypeStory]
	
	// Health criteria
	if totalItems > 150 {
		return "AT_RISK"
	}
	if wipCount > 20 || leadTime > 60 {
		return "WARNING"
	}
	if totalItems <= 100 && wipCount <= 10 && leadTime <= 30 {
		return "HEALTHY"
	}
	
	return "AVERAGE"
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Request/Response Types

type CreateItemRequest struct {
	Type        model.ItemType
	Title       string
	Description string
	ParentID    *uuid.UUID
	StoryPoints int
	Tags        []string
	Assignee    string
}

type UpdateItemRequest struct {
	Title       *string
	Description *string
	Status      *model.ItemStatus
	ParentID    *uuid.UUID
	StoryPoints *int
	Priority    *int
	Assignee    *string
	Tags        *[]string
}

type ReorderRequest struct {
	ItemID      uuid.UUID
	NewPriority int
}

type BacklogMetrics struct {
	TotalItems           int     `json:"totalItems"`
	EpicCount            int     `json:"epicCount"`
	FeatureCount         int     `json:"featureCount"`
	StoryCount           int     `json:"storyCount"`
	AverageAge           float64 `json:"averageAge"`
	WIPCount             int     `json:"wipCount"`
	LeadTimeDays         float64 `json:"leadTimeDays"`
	ThroughputLast30Days int     `json:"throughputLast30Days"`
	IcebergRatio         float64 `json:"icebergRatio"`
	HealthStatus         string  `json:"healthStatus"`
}

type listCacheResult struct {
	Items      []*model.BacklogItem
	TotalCount int64
}
