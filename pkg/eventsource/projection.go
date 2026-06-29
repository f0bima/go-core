package eventsource

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/f0bima/go-core/pkg/cqrs"
)

// Projection defines a read model that is built from events.
type Projection interface {
	Handle(event cqrs.DomainEvent) error
	GetName() string
}

// ProjectionBuilder builds and updates projections from event streams.
type ProjectionBuilder struct {
	projections []Projection
	mu          sync.RWMutex
}

// NewProjectionBuilder creates a new projection builder.
func NewProjectionBuilder() *ProjectionBuilder {
	return &ProjectionBuilder{
		projections: make([]Projection, 0),
	}
}

// Register adds a projection to the builder.
func (pb *ProjectionBuilder) Register(projection Projection) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.projections = append(pb.projections, projection)
}

// ProcessEvent updates all projections with an event.
func (pb *ProjectionBuilder) ProcessEvent(event cqrs.DomainEvent) error {
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	var errors []error

	for _, projection := range pb.projections {
		if err := projection.Handle(event); err != nil {
			slog.Error("Projection handler failed",
				"projection", projection.GetName(),
				"event_type", event.EventType,
				"error", err,
			)
			errors = append(errors, fmt.Errorf("projection %s: %w", projection.GetName(), err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%d projection(s) failed: %v", len(errors), errors[0])
	}

	return nil
}

// ProcessEvents processes multiple events in order.
func (pb *ProjectionBuilder) ProcessEvents(events []cqrs.DomainEvent) error {
	for _, event := range events {
		if err := pb.ProcessEvent(event); err != nil {
			return err
		}
	}
	return nil
}

// Rebuild rebuilds a projection from event history.
func (pb *ProjectionBuilder) Rebuild(ctx context.Context, eventStore *cqrs.EventStore, projectionName string) error {
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	// Find the projection
	var target Projection
	for _, p := range pb.projections {
		if p.GetName() == projectionName {
			target = p
			break
		}
	}

	if target == nil {
		return fmt.Errorf("projection not found: %s", projectionName)
	}

	// Get all events
	events, err := eventStore.GetGlobalStream(ctx, "", 0)
	if err != nil {
		return fmt.Errorf("get global event stream: %w", err)
	}

	slog.Info("Rebuilding projection",
		"projection", projectionName,
		"total_events", len(events),
	)

	// Process all events
	for i, event := range events {
		if err := target.Handle(event); err != nil {
			slog.Error("Failed to rebuild projection",
				"projection", projectionName,
				"event_index", i,
				"event_type", event.EventType,
				"error", err,
			)
			return fmt.Errorf("rebuild projection at event %d: %w", i, err)
		}
	}

	slog.Info("Projection rebuilt successfully",
		"projection", projectionName,
		"events_processed", len(events),
	)

	return nil
}

// GetAllProjections returns all registered projection names.
func (pb *ProjectionBuilder) GetAllProjections() []string {
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	names := make([]string, len(pb.projections))
	for i, p := range pb.projections {
		names[i] = p.GetName()
	}

	return names
}
