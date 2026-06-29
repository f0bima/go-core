package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ChoreographyEvent represents an event in a choreography-based saga.
type ChoreographyEvent struct {
	SagaID        uuid.UUID       `json:"saga_id"`
	StepName      string          `json:"step_name"`
	Status        string          `json:"status"` // started, completed, failed, compensated
	Payload       json.RawMessage `json:"payload,omitempty"`
	Error         string          `json:"error,omitempty"`
	Timestamp     time.Time       `json:"timestamp"`
	NextStep      string          `json:"next_step,omitempty"`
	Compensating  bool            `json:"compensating,omitempty"`
}

// SagaStepHandler handles a specific step in a choreography saga.
type SagaStepHandler struct {
	StepName   string
	Handle     func(ctx context.Context, event *ChoreographyEvent) (*ChoreographyEvent, error)
	Compensate func(ctx context.Context, event *ChoreographyEvent) (*ChoreographyEvent, error)
}

// ChoreographySaga manages event-driven saga orchestration.
type ChoreographySaga struct {
	Name     string
	Handlers map[string]*SagaStepHandler
	Store    SagaStore
	mu       sync.RWMutex
}

// SagaStore persists saga state for recovery and monitoring.
type SagaStore interface {
	SaveSagaState(ctx context.Context, sagaID uuid.UUID, state []byte) error
	GetSagaState(ctx context.Context, sagaID uuid.UUID) ([]byte, error)
	GetIncompleteSagas(ctx context.Context) ([]uuid.UUID, error)
}

// NewChoreographySaga creates a new choreography-based saga.
func NewChoreographySaga(name string, store SagaStore) *ChoreographySaga {
	return &ChoreographySaga{
		Name:     name,
		Handlers: make(map[string]*SagaStepHandler),
		Store:    store,
	}
}

// AddHandler registers a handler for a saga step.
func (s *ChoreographySaga) AddHandler(handler *SagaStepHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Handlers[handler.StepName] = handler
}

// StartSaga begins a new saga instance.
func (s *ChoreographySaga) StartSaga(ctx context.Context, initialPayload interface{}) (uuid.UUID, error) {
	sagaID := uuid.New()

	event := &ChoreographyEvent{
		SagaID:    sagaID,
		StepName:  "init",
		Status:    "started",
		Timestamp: time.Now(),
	}

	if initialPayload != nil {
		payloadBytes, err := json.Marshal(initialPayload)
		if err != nil {
			return uuid.Nil, fmt.Errorf("marshal initial payload: %w", err)
		}
		event.Payload = payloadBytes
	}

	// Save initial state
	stateBytes, err := json.Marshal(event)
	if err != nil {
		return uuid.Nil, fmt.Errorf("marshal saga state: %w", err)
	}

	if err := s.Store.SaveSagaState(ctx, sagaID, stateBytes); err != nil {
		return uuid.Nil, fmt.Errorf("save saga state: %w", err)
	}

	slog.Info("Choreography saga started",
		"saga_name", s.Name,
		"saga_id", sagaID,
	)

	return sagaID, nil
}

// HandleEvent processes a saga event and executes the appropriate step.
func (s *ChoreographySaga) HandleEvent(ctx context.Context, event *ChoreographyEvent) (*ChoreographyEvent, error) {
	s.mu.RLock()
	handler, exists := s.Handlers[event.StepName]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no handler for step: %s", event.StepName)
	}

	// Execute the step
	resultEvent, err := handler.Handle(ctx, event)
	if err != nil {
		slog.Error("Saga step handler failed",
			"saga_name", s.Name,
			"saga_id", event.SagaID,
			"step_name", event.StepName,
			"error", err,
		)

		// Mark as failed
		failedEvent := &ChoreographyEvent{
			SagaID:    event.SagaID,
			StepName:  event.StepName,
			Status:    "failed",
			Error:     err.Error(),
			Timestamp: time.Now(),
		}

		// Save failed state
		if storeErr := s.saveEventState(ctx, failedEvent); storeErr != nil {
			slog.Error("Failed to save saga state", "error", storeErr)
		}

		return failedEvent, err
	}

	// Save completed state
	if err := s.saveEventState(ctx, resultEvent); err != nil {
		slog.Error("Failed to save saga state", "error", err)
	}

	slog.Info("Saga step completed",
		"saga_name", s.Name,
		"saga_id", event.SagaID,
		"step_name", event.StepName,
		"next_step", resultEvent.NextStep,
	)

	return resultEvent, nil
}

// Compensate executes compensation for a failed step.
func (s *ChoreographySaga) Compensate(ctx context.Context, event *ChoreographyEvent) (*ChoreographyEvent, error) {
	s.mu.RLock()
	handler, exists := s.Handlers[event.StepName]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no handler for step: %s", event.StepName)
	}

	if handler.Compensate == nil {
		slog.Warn("No compensation defined for step",
			"saga_name", s.Name,
			"step_name", event.StepName,
		)
		return event, nil
	}

	event.Compensating = true
	resultEvent, err := handler.Compensate(ctx, event)
	if err != nil {
		slog.Error("Saga compensation failed",
			"saga_name", s.Name,
			"saga_id", event.SagaID,
			"step_name", event.StepName,
			"error", err,
		)
		return nil, err
	}

	resultEvent.Status = "compensated"
	resultEvent.Timestamp = time.Now()

	if err := s.saveEventState(ctx, resultEvent); err != nil {
		slog.Error("Failed to save compensation state", "error", err)
	}

	slog.Info("Saga step compensated",
		"saga_name", s.Name,
		"saga_id", event.SagaID,
		"step_name", event.StepName,
	)

	return resultEvent, nil
}

// saveEventState persists the event state to the store.
func (s *ChoreographySaga) saveEventState(ctx context.Context, event *ChoreographyEvent) error {
	stateBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event state: %w", err)
	}

	return s.Store.SaveSagaState(ctx, event.SagaID, stateBytes)
}
