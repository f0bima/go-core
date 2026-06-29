package cqrs

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DomainEvent represents an event in the event store.
type DomainEvent struct {
	ID          uuid.UUID       `json:"id"`
	AggregateID uuid.UUID       `json:"aggregate_id"`
	EventType   string          `json:"event_type"`
	Payload     json.RawMessage `json:"payload"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	Version     int             `json:"version"`
	Timestamp   time.Time       `json:"timestamp"`
}

// EventStore stores and retrieves domain events.
type EventStore struct {
	mu     sync.RWMutex
	events map[uuid.UUID][]DomainEvent // aggregate_id -> events
}

// NewEventStore creates a new event store.
func NewEventStore() *EventStore {
	return &EventStore{
		events: make(map[uuid.UUID][]DomainEvent),
	}
}

// Append adds events to the store (optimistic concurrency).
func (es *EventStore) Append(ctx context.Context, aggregateID uuid.UUID, expectedVersion int, newEvents []DomainEvent) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	currentEvents := es.events[aggregateID]
	currentVersion := len(currentEvents)

	if expectedVersion != currentVersion {
		return fmt.Errorf("concurrency conflict: expected version %d, got %d", expectedVersion, currentVersion)
	}

	// Append new events
	for i, event := range newEvents {
		event.ID = uuid.New()
		event.AggregateID = aggregateID
		event.Version = currentVersion + i + 1
		event.Timestamp = time.Now()
		currentEvents = append(currentEvents, event)
	}

	es.events[aggregateID] = currentEvents

	return nil
}

// GetEvents retrieves events for an aggregate.
func (es *EventStore) GetEvents(ctx context.Context, aggregateID uuid.UUID, afterVersion int) ([]DomainEvent, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	events := es.events[aggregateID]
	if afterVersion >= 0 {
		var filtered []DomainEvent
		for _, event := range events {
			if event.Version > afterVersion {
				filtered = append(filtered, event)
			}
		}
		return filtered, nil
	}

	return events, nil
}

// GetAggregateStream returns all events for an aggregate in order.
func (es *EventStore) GetAggregateStream(ctx context.Context, aggregateID uuid.UUID) ([]DomainEvent, error) {
	return es.GetEvents(ctx, aggregateID, -1)
}

// GetGlobalStream returns events across all aggregates, optionally filtered by type.
func (es *EventStore) GetGlobalStream(ctx context.Context, eventType string, limit int) ([]DomainEvent, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	var allEvents []DomainEvent
	for _, events := range es.events {
		for _, event := range events {
			if eventType == "" || event.EventType == eventType {
				allEvents = append(allEvents, event)
			}
		}
	}

	// Sort by timestamp
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].Timestamp.Before(allEvents[j].Timestamp)
	})

	if limit > 0 && len(allEvents) > limit {
		return allEvents[:limit], nil
	}

	return allEvents, nil
}

// GetLatestVersion returns the latest version for an aggregate.
func (es *EventStore) GetLatestVersion(ctx context.Context, aggregateID uuid.UUID) int {
	es.mu.RLock()
	defer es.mu.RUnlock()

	events := es.events[aggregateID]
	if len(events) == 0 {
		return 0
	}

	return events[len(events)-1].Version
}

// NewDomainEvent creates a new domain event.
func NewDomainEvent(eventType string, aggregateID uuid.UUID, payload interface{}) (DomainEvent, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return DomainEvent{}, fmt.Errorf("marshal payload: %w", err)
	}

	return DomainEvent{
		EventType:   eventType,
		AggregateID: aggregateID,
		Payload:     payloadBytes,
		Timestamp:   time.Now(),
	}, nil
}
