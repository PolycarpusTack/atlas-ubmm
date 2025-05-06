// services/backlog-service/internal/adapters/eventbus/kafka.go

package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"go.uber.org/zap"

	"github.com/ubmm/backlog-service/internal/config"
	"github.com/ubmm/backlog-service/internal/domain/event"
)

// KafkaAdapter implements the event publisher interface
type KafkaAdapter struct {
	producer *kafka.Producer
	logger   *zap.Logger
}

// NewKafkaAdapter creates a new Kafka adapter
func NewKafkaAdapter(cfg config.KafkaConfig, logger *zap.Logger) (*KafkaAdapter, error) {
	// Create Kafka producer
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":        cfg.BootstrapServers,
		"client.id":                cfg.ClientID,
		"acks":                     "all",
		"retries":                  10,
		"retry.backoff.ms":         250,
		"queue.buffering.max.ms":   100,
		"queue.buffering.max.kbytes": 1024 * 16,
		"batch.size":               16384,
		"linger.ms":                10,
		"request.timeout.ms":       30000,
		"message.timeout.ms":       60000,
		
		// Enable idempotent producer for exactly-once semantics
		"enable.idempotence":       true,
		
		// Security settings
		"security.protocol":        cfg.SecurityProtocol,
		"sasl.mechanisms":          cfg.SASLMechanism,
		"sasl.username":            cfg.SASLUsername,
		"sasl.password":            cfg.SASLPassword,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	// Start event handling goroutine
	go func() {
		for e := range producer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					logger.Error("Failed to deliver message",
						zap.String("topic", *ev.TopicPartition.Topic),
						zap.String("key", string(ev.Key)),
						zap.Error(ev.TopicPartition.Error))
				} else {
					logger.Debug("Message delivered",
						zap.String("topic", *ev.TopicPartition.Topic),
						zap.String("key", string(ev.Key)),
						zap.Int32("partition", ev.TopicPartition.Partition),
						zap.Int64("offset", int64(ev.TopicPartition.Offset)))
				}
			default:
				logger.Debug("Ignored event", zap.String("type", fmt.Sprintf("%T", e)))
			}
		}
	}()

	return &KafkaAdapter{
		producer: producer,
		logger:   logger,
	}, nil
}

// Close closes the Kafka producer
func (a *KafkaAdapter) Close() error {
	// Wait for any outstanding messages to be delivered
	a.producer.Flush(15000) // 15 seconds timeout
	a.producer.Close()
	return nil
}

// Publish publishes an event to Kafka
func (a *KafkaAdapter) Publish(ctx context.Context, topic string, event interface{}) error {
	// Marshal event to JSON
	jsonBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Extract key from event if available
	var key []byte
	if e, ok := event.(interface{ GetID() string }); ok {
		key = []byte(e.GetID())
	} else {
		// Generate a timestamp-based key if no ID is available
		key = []byte(fmt.Sprintf("%d", time.Now().UnixNano()))
	}

	// Deliver message to Kafka
	message := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Key:   key,
		Value: jsonBytes,
		// Add headers if needed
		Headers: []kafka.Header{
			{
				Key:   "content-type",
				Value: []byte("application/json"),
			},
			{
				Key:   "source",
				Value: []byte("backlog-service"),
			},
			{
				Key:   "timestamp",
				Value: []byte(fmt.Sprintf("%d", time.Now().Unix())),
			},
		},
	}

	// Publish message
	err = a.producer.Produce(message, nil)
	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	return nil
}

// Implements the KafkaProducer interface for event.KafkaPublisher
type KafkaProducerAdapter struct {
	producer *kafka.Producer
	logger   *zap.Logger
}

// NewKafkaProducerAdapter creates a new Kafka producer adapter
func NewKafkaProducerAdapter(producer *kafka.Producer, logger *zap.Logger) event.KafkaProducer {
	return &KafkaProducerAdapter{
		producer: producer,
		logger:   logger,
	}
}

// Send sends a message to Kafka
func (a *KafkaProducerAdapter) Send(ctx context.Context, topic string, key string, value []byte) error {
	message := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Key:   []byte(key),
		Value: value,
		Headers: []kafka.Header{
			{
				Key:   "content-type",
				Value: []byte("application/json"),
			},
			{
				Key:   "source",
				Value: []byte("backlog-service"),
			},
			{
				Key:   "timestamp",
				Value: []byte(fmt.Sprintf("%d", time.Now().Unix())),
			},
		},
	}

	// Use context deadline if available
	deadline, ok := ctx.Deadline()
	if ok {
		timeout := time.Until(deadline)
		if timeout <= 0 {
			return fmt.Errorf("context deadline exceeded")
		}

		// Set message delivery timeout
		if timeout > 60*time.Second {
			timeout = 60 * time.Second // Cap at 60 seconds
		}
		message.Headers = append(message.Headers, kafka.Header{
			Key:   "timeout",
			Value: []byte(fmt.Sprintf("%d", int(timeout.Milliseconds()))),
		})
	}

	// Produce the message
	err := a.producer.Produce(message, nil)
	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	return nil
}

// Close closes the Kafka producer
func (a *KafkaProducerAdapter) Close() error {
	a.producer.Flush(15000) // 15 seconds timeout
	a.producer.Close()
	return nil
}

// KafkaConsumer provides consumer functionality
type KafkaConsumer struct {
	consumer *kafka.Consumer
	logger   *zap.Logger
}

// NewKafkaConsumer creates a new Kafka consumer
func NewKafkaConsumer(cfg config.KafkaConfig, consumerGroup string, logger *zap.Logger) (*KafkaConsumer, error) {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":       cfg.BootstrapServers,
		"group.id":                consumerGroup,
		"auto.offset.reset":       "earliest",
		"enable.auto.commit":      false,
		"auto.commit.interval.ms": 5000,
		"session.timeout.ms":      30000,
		"max.poll.interval.ms":    300000,
		
		// Security settings
		"security.protocol":       cfg.SecurityProtocol,
		"sasl.mechanisms":         cfg.SASLMechanism,
		"sasl.username":           cfg.SASLUsername,
		"sasl.password":           cfg.SASLPassword,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	return &KafkaConsumer{
		consumer: consumer,
		logger:   logger,
	}, nil
}

// Close closes the Kafka consumer
func (c *KafkaConsumer) Close() error {
	return c.consumer.Close()
}

// Subscribe subscribes to topics
func (c *KafkaConsumer) Subscribe(topics []string) error {
	return c.consumer.SubscribeTopics(topics, nil)
}

// MessageHandler defines a function to handle Kafka messages
type MessageHandler func(message *kafka.Message) error

// ConsumeMessages starts consuming messages
func (c *KafkaConsumer) ConsumeMessages(ctx context.Context, handler MessageHandler) error {
	// Start consuming in a loop
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Poll for messages with a timeout
			msg, err := c.consumer.ReadMessage(100 * time.Millisecond)
			if err != nil {
				if err.(kafka.Error).Code() == kafka.ErrTimedOut {
					// Timeout is not an error, just continue
					continue
				}
				c.logger.Error("Failed to read message", zap.Error(err))
				continue
			}

			// Process the message
			err = handler(msg)
			if err != nil {
				c.logger.Error("Failed to process message",
					zap.String("topic", *msg.TopicPartition.Topic),
					zap.String("key", string(msg.Key)),
					zap.Error(err))
				// Continue processing other messages
				continue
			}

			// Commit offset for the processed message
			_, err = c.consumer.CommitMessage(msg)
			if err != nil {
				c.logger.Error("Failed to commit offset",
					zap.String("topic", *msg.TopicPartition.Topic),
					zap.Int32("partition", msg.TopicPartition.Partition),
					zap.Int64("offset", int64(msg.TopicPartition.Offset)),
					zap.Error(err))
			}
		}
	}
}
