package event

import (
	"database/sql"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v2/pkg/amqp"
	watermillsql "github.com/ThreeDotsLabs/watermill-sql/v3/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/components/forwarder"
	"github.com/ThreeDotsLabs/watermill/message"
)

// NewAMQPPublisher creates a new RabbitMQ Publisher
func NewAMQPPublisher(amqpURI string, logger watermill.LoggerAdapter) (*amqp.Publisher, error) {
	amqpConfig := amqp.NewDurablePubSubConfig(
		amqpURI,
		amqp.GenerateQueueNameTopicNameWithSuffix("queue"),
	)
	return amqp.NewPublisher(amqpConfig, logger)
}

// NewAMQPSubscriber creates a new RabbitMQ Subscriber
func NewAMQPSubscriber(amqpURI string, logger watermill.LoggerAdapter) (*amqp.Subscriber, error) {
	amqpConfig := amqp.NewDurablePubSubConfig(
		amqpURI,
		amqp.GenerateQueueNameTopicNameWithSuffix("queue"),
	)
	return amqp.NewSubscriber(amqpConfig, logger)
}

// NewSQLOutboxPublisher creates a publisher that writes to the outbox table in PostgreSQL
func NewSQLOutboxPublisher(db *sql.DB, logger watermill.LoggerAdapter) (*watermillsql.Publisher, error) {
	return watermillsql.NewPublisher(
		db,
		watermillsql.PublisherConfig{
			SchemaAdapter: watermillsql.DefaultPostgreSQLSchema{},
			AutoInitializeSchema: true,
		},
		logger,
	)
}

// NewOutboxForwarder creates a forwarder that reads from the SQL outbox table and publishes to AMQP
func NewOutboxForwarder(
	sqlSubscriber *watermillsql.Subscriber,
	amqpPublisher message.Publisher,
	logger watermill.LoggerAdapter,
) (*forwarder.Forwarder, error) {
	return forwarder.NewForwarder(
		sqlSubscriber,
		amqpPublisher,
		logger,
		forwarder.Config{
			ForwarderTopic: "events_to_forward",
			Middlewares: []message.HandlerMiddleware{
				// Add any middlewares here if needed
			},
		},
	)
}

// NewSQLOutboxSubscriber creates a subscriber that reads from the outbox table
func NewSQLOutboxSubscriber(db *sql.DB, logger watermill.LoggerAdapter) (*watermillsql.Subscriber, error) {
	return watermillsql.NewSubscriber(
		db,
		watermillsql.SubscriberConfig{
			SchemaAdapter:    watermillsql.DefaultPostgreSQLSchema{},
			OffsetsAdapter:   watermillsql.DefaultPostgreSQLOffsetsAdapter{},
			InitializeSchema: true,
		},
		logger,
	)
}
