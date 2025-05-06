// services/backlog-service/internal/domain/event/publisher.go

package event

import (
	"context"
)

// Publisher defines the interface for publishing events
type Publisher interface {
	// Publish publishes an event to the specified topic
	Publish(ctx context.Context, topic string, event interface{}) error
}

// KafkaPublisher implements the Publisher interface using Kafka
type KafkaPublisher struct {
	producer KafkaProducer
}

// KafkaProducer defines the interface for Kafka producer
type KafkaProducer interface {
	// Send sends a message to Kafka
	Send(ctx context.Context, topic string, key string, value []byte) error
	// Close closes the producer
	Close() error
}

// NewKafkaPublisher creates a new Kafka publisher
func NewKafkaPublisher(producer KafkaProducer) *KafkaPublisher {
	return &KafkaPublisher{
		producer: producer,
	}
}

// Publish publishes an event to Kafka
func (p *KafkaPublisher) Publish(ctx context.Context, topic string, event interface{}) error {
	// Convert event to JSON
	jsonBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Extract event ID for key if available
	key := ""
	if e, ok := event.(interface{ GetID() string }); ok {
		key = e.GetID()
	}

	// Send to Kafka
	return p.producer.Send(ctx, topic, key, jsonBytes)
}

// NoopPublisher implements the Publisher interface with no-op
type NoopPublisher struct{}

// NewNoopPublisher creates a new no-op publisher
func NewNoopPublisher() *NoopPublisher {
	return &NoopPublisher{}
}

// Publish does nothing
func (p *NoopPublisher) Publish(ctx context.Context, topic string, event interface{}) error {
	// No-op implementation
	return nil
}
