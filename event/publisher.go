package event

import (
	"database/sql"
	"errors"

	"github.com/ThreeDotsLabs/watermill"
	watermillsql "github.com/ThreeDotsLabs/watermill-sql/v3/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/components/forwarder"
	"github.com/ThreeDotsLabs/watermill/message"
	"gorm.io/gorm"
)

// PublishWithinTx creates a temporary publisher scoped to the GORM transaction
// and publishes the message to the SQL Outbox table safely.
func PublishWithinTx(tx *gorm.DB, topic string, msg *message.Message, logger watermill.LoggerAdapter) error {
	if tx == nil || tx.Statement == nil || tx.Statement.ConnPool == nil {
		return errors.New("invalid gorm transaction")
	}

	sqlTx, ok := tx.Statement.ConnPool.(*sql.Tx)
	if !ok {
		return errors.New("could not extract sql.Tx from gorm DB")
	}

	sqlPublisher, err := watermillsql.NewPublisher(
		sqlTx,
		watermillsql.PublisherConfig{
			SchemaAdapter:        watermillsql.DefaultPostgreSQLSchema{},
			AutoInitializeSchema: false,
		},
		logger,
	)
	if err != nil {
		return err
	}

	forwarderPub := forwarder.NewPublisher(sqlPublisher, forwarder.PublisherConfig{
		ForwarderTopic: "events_to_forward",
	})

	return forwarderPub.Publish(topic, msg)
}
