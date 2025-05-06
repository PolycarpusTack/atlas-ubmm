// services/backlog-service/internal/domain/repository/repository.go

package repository

import (
	"context"

	"github.com/google/uuid"
	
	"github.com/ubmm/backlog-service/internal/domain/model"
)

// Repository defines the interface for backlog item persistence
type BacklogRepository interface {
	// Create stores a new backlog item
	Create(ctx context.Context, item *model.BacklogItem) error
	
	// GetByID retrieves a backlog item by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*model.BacklogItem, error)
	
	// GetByExternalID retrieves a backlog item by its external ID
	GetByExternalID(ctx context.Context, system, externalID string) (*model.BacklogItem, error)
	
	// Update updates an existing backlog item
	Update(ctx context.Context, item *model.BacklogItem) error
	
	// Delete deletes a backlog item by its ID
	Delete(ctx context.Context, id uuid.UUID) error
	
	// List retrieves backlog items with pagination
	List(ctx context.Context, filter BacklogFilter) ([]*model.BacklogItem, int64, error)
	
	// GetChildren retrieves all children of a backlog item
	GetChildren(ctx context.Context, parentID uuid.UUID) ([]*model.BacklogItem, error)
	
	// UpdatePriorities updates the priorities of multiple items in a batch
	UpdatePriorities(ctx context.Context, itemPriorities map[uuid.UUID]int) error
}

// BacklogFilter defines filters for listing backlog items
type BacklogFilter struct {
	Types       []model.ItemType
	Statuses    []model.ItemStatus
	Tags        []string
	ParentID    *uuid.UUID
	Assignee    string
	SearchQuery string
	Limit       int
	Offset      int
	SortBy      string
	SortOrder   string
}

// EventRepository defines the interface for event sourcing
type EventRepository interface {
	// StoreEvent stores a domain event
	StoreEvent(ctx context.Context, event interface{}) error
	
	// GetEventsByItemID retrieves events for a specific backlog item
	GetEventsByItemID(ctx context.Context, itemID uuid.UUID) ([]interface{}, error)
	
	// ReplayEvents replays events to reconstruct state
	ReplayEvents(ctx context.Context, itemID uuid.UUID) (*model.BacklogItem, error)
}

// MetricsRepository defines the interface for backlog metrics
type MetricsRepository interface {
	// GetBacklogSize retrieves the current backlog size metrics
	GetBacklogSize(ctx context.Context) (map[model.ItemType]int, error)
	
	// GetItemAge retrieves age metrics for backlog items
	GetItemAge(ctx context.Context, status model.ItemStatus) (map[model.ItemType]float64, error)
	
	// GetWIPCounts retrieves work-in-progress counts
	GetWIPCounts(ctx context.Context) (int, error)
	
	// GetLeadTime retrieves lead time metrics
	GetLeadTime(ctx context.Context, timeWindowDays int) (float64, error)
	
	// GetThroughput retrieves throughput metrics
	GetThroughput(ctx context.Context, timeWindowDays int) (int, error)
}
