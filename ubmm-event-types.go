// services/backlog-service/internal/domain/event/events.go

package event

import (
	"time"

	"github.com/google/uuid"
	
	"github.com/ubmm/backlog-service/internal/domain/model"
)

// EventType defines the type of event
type EventType string

const (
	// EventTypeItemCreated represents an item created event
	EventTypeItemCreated EventType = "ITEM_CREATED"
	// EventTypeItemUpdated represents an item updated event
	EventTypeItemUpdated EventType = "ITEM_UPDATED"
	// EventTypeItemDeleted represents an item deleted event
	EventTypeItemDeleted EventType = "ITEM_DELETED"
	// EventTypeItemsReordered represents items reordered event
	EventTypeItemsReordered EventType = "ITEMS_REORDERED"
	// EventTypeExternalIDSet represents an external ID set event
	EventTypeExternalIDSet EventType = "EXTERNAL_ID_SET"
)

// Event defines the base event structure
type Event struct {
	ID        uuid.UUID `json:"id"`
	Type      EventType `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Version   int       `json:"version"`
}

// ItemCreatedEvent represents an event when a backlog item is created
type ItemCreatedEvent struct {
	Event
	ItemID uuid.UUID          `json:"itemId"`
	Item   *model.BacklogItem `json:"item"`
}

// ItemUpdatedEvent represents an event when a backlog item is updated
type ItemUpdatedEvent struct {
	Event
	ItemID uuid.UUID          `json:"itemId"`
	Item   *model.BacklogItem `json:"item"`
}

// ItemDeletedEvent represents an event when a backlog item is deleted
type ItemDeletedEvent struct {
	Event
	ItemID uuid.UUID          `json:"itemId"`
	Item   *model.BacklogItem `json:"item"`
}

// ItemsReorderedEvent represents an event when backlog items are reordered
type ItemsReorderedEvent struct {
	Event
	ItemPriorities map[uuid.UUID]int `json:"itemPriorities"`
}

// ExternalIDSetEvent represents an event when an external ID is set for an item
type ExternalIDSetEvent struct {
	Event
	ItemID     uuid.UUID `json:"itemId"`
	System     string    `json:"system"`
	ExternalID string    `json:"externalId"`
}

// NewBaseEvent creates a new base event
func NewBaseEvent(eventType EventType) Event {
	return Event{
		ID:        uuid.New(),
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Version:   1,
	}
}

// NewItemCreatedEvent creates a new item created event
func NewItemCreatedEvent(itemID uuid.UUID, item *model.BacklogItem) *ItemCreatedEvent {
	return &ItemCreatedEvent{
		Event:  NewBaseEvent(EventTypeItemCreated),
		ItemID: itemID,
		Item:   item,
	}
}

// NewItemUpdatedEvent creates a new item updated event
func NewItemUpdatedEvent(itemID uuid.UUID, item *model.BacklogItem) *ItemUpdatedEvent {
	return &ItemUpdatedEvent{
		Event:  NewBaseEvent(EventTypeItemUpdated),
		ItemID: itemID,
		Item:   item,
	}
}

// NewItemDeletedEvent creates a new item deleted event
func NewItemDeletedEvent(itemID uuid.UUID, item *model.BacklogItem) *ItemDeletedEvent {
	return &ItemDeletedEvent{
		Event:  NewBaseEvent(EventTypeItemDeleted),
		ItemID: itemID,
		Item:   item,
	}
}

// NewItemsReorderedEvent creates a new items reordered event
func NewItemsReorderedEvent(itemPriorities map[uuid.UUID]int) *ItemsReorderedEvent {
	return &ItemsReorderedEvent{
		Event:          NewBaseEvent(EventTypeItemsReordered),
		ItemPriorities: itemPriorities,
	}
}

// NewExternalIDSetEvent creates a new external ID set event
func NewExternalIDSetEvent(itemID uuid.UUID, system, externalID string) *ExternalIDSetEvent {
	return &ExternalIDSetEvent{
		Event:      NewBaseEvent(EventTypeExternalIDSet),
		ItemID:     itemID,
		System:     system,
		ExternalID: externalID,
	}
}
