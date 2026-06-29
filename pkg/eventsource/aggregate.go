package eventsource

import (
	"context"
	"fmt"

	"github.com/f0bima/go-core/pkg/cqrs"
	"github.com/google/uuid"
)

// Aggregate is the base for event-sourced aggregates.
type Aggregate struct {
	ID      uuid.UUID
	Version int
	Changes []cqrs.DomainEvent
}

// NewAggregate creates a new aggregate.
func NewAggregate(id uuid.UUID) *Aggregate {
	return &Aggregate{
		ID:      id,
		Version: 0,
		Changes: make([]cqrs.DomainEvent, 0),
	}
}

// LoadFromHistory rebuilds aggregate state from event history.
func (a *Aggregate) LoadFromHistory(events []cqrs.DomainEvent) error {
	for _, event := range events {
		if err := a.ApplyEvent(event); err != nil {
			return fmt.Errorf("apply event %s (version %d): %w", event.EventType, event.Version, err)
		}
		a.Version = event.Version
	}
	return nil
}

// ApplyEvent applies a single event to the aggregate (to be overridden by concrete aggregates).
func (a *Aggregate) ApplyEvent(event cqrs.DomainEvent) error {
	// Base implementation - concrete aggregates should override
	return nil
}

// ApplyChange records a new change/event.
func (a *Aggregate) ApplyChange(eventType string, payload interface{}) error {
	event, err := cqrs.NewDomainEvent(eventType, a.ID, payload)
	if err != nil {
		return err
	}

	event.Version = a.Version + 1
	a.Changes = append(a.Changes, event)
	a.Version++

	// Apply to current state
	return a.ApplyEvent(event)
}

// GetUncommittedEvents returns events that haven't been persisted yet.
func (a *Aggregate) GetUncommittedEvents() []cqrs.DomainEvent {
	return a.Changes
}

// ClearChanges clears uncommitted changes after persistence.
func (a *Aggregate) ClearChanges() {
	a.Changes = make([]cqrs.DomainEvent, 0)
}

// AggregateRepository handles aggregate persistence.
type AggregateRepository struct {
	eventStore *cqrs.EventStore
}

// NewAggregateRepository creates a new aggregate repository.
func NewAggregateRepository(eventStore *cqrs.EventStore) *AggregateRepository {
	return &AggregateRepository{
		eventStore: eventStore,
	}
}

// Save persists uncommitted events from an aggregate.
func (r *AggregateRepository) Save(ctx context.Context, aggregate *Aggregate) error {
	changes := aggregate.GetUncommittedEvents()
	if len(changes) == 0 {
		return nil
	}

	expectedVersion := aggregate.Version - len(changes)

	if err := r.eventStore.Append(ctx, aggregate.ID, expectedVersion, changes); err != nil {
		return fmt.Errorf("save aggregate events: %w", err)
	}

	aggregate.ClearChanges()
	return nil
}

// Load rebuilds an aggregate from its event history.
func (r *AggregateRepository) Load(ctx context.Context, id uuid.UUID, aggregate *Aggregate) error {
	events, err := r.eventStore.GetAggregateStream(ctx, id)
	if err != nil {
		return fmt.Errorf("load aggregate events: %w", err)
	}

	if len(events) == 0 {
		return ErrAggregateNotFound
	}

	if err := aggregate.LoadFromHistory(events); err != nil {
		return fmt.Errorf("rebuild aggregate: %w", err)
	}

	return nil
}

// ErrAggregateNotFound is returned when an aggregate doesn't exist.
var ErrAggregateNotFound = fmt.Errorf("aggregate not found")
