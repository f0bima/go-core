package saga

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Step defines a single step in a saga with forward and compensating actions.
type Step struct {
	Name        string
	Action      func(ctx context.Context, sagaCtx *SagaContext) error
	Compensate  func(ctx context.Context, sagaCtx *SagaContext) error
	Description string
}

// SagaContext holds shared data across all saga steps.
type SagaContext struct {
	Data      map[string]interface{}
	StartTime time.Time
	Steps     []string
	Errors    []error
}

// NewSagaContext creates a new saga context.
func NewSagaContext() *SagaContext {
	return &SagaContext{
		Data:      make(map[string]interface{}),
		StartTime: time.Now(),
		Steps:     make([]string, 0),
		Errors:    make([]error, 0),
	}
}

// Set stores data in the saga context.
func (ctx *SagaContext) Set(key string, value interface{}) {
	ctx.Data[key] = value
}

// Get retrieves data from the saga context.
func (ctx *SagaContext) Get(key string) (interface{}, bool) {
	value, exists := ctx.Data[key]
	return value, exists
}

// GetString retrieves a string value from the saga context.
func (ctx *SagaContext) GetString(key string) (string, bool) {
	value, exists := ctx.Data[key]
	if !exists {
		return "", false
	}
	str, ok := value.(string)
	return str, ok
}

// RecordStep records a completed step.
func (ctx *SagaContext) RecordStep(stepName string) {
	ctx.Steps = append(ctx.Steps, stepName)
}

// RecordError records an error.
func (ctx *SagaContext) RecordError(err error) {
	ctx.Errors = append(ctx.Errors, err)
}

// Saga orchestrates a sequence of steps with compensation on failure.
type Saga struct {
	Name    string
	Steps   []Step
	OnError func(ctx context.Context, sagaCtx *SagaContext, stepIndex int, err error) error
}

// NewSaga creates a new saga orchestrator.
func NewSaga(name string) *Saga {
	return &Saga{
		Name:  name,
		Steps: make([]Step, 0),
	}
}

// AddStep adds a step to the saga.
func (s *Saga) AddStep(name string, action func(ctx context.Context, sagaCtx *SagaContext) error, compensate func(ctx context.Context, sagaCtx *SagaContext) error) *Saga {
	s.Steps = append(s.Steps, Step{
		Name:       name,
		Action:     action,
		Compensate: compensate,
	})
	return s
}

// SetErrorHandler sets a custom error handler for the saga.
func (s *Saga) SetErrorHandler(handler func(ctx context.Context, sagaCtx *SagaContext, stepIndex int, err error) error) *Saga {
	s.OnError = handler
	return s
}

// Execute runs all saga steps in order. If any step fails, it compensates completed steps in reverse order.
func (s *Saga) Execute(ctx context.Context, sagaCtx *SagaContext) error {
	if sagaCtx == nil {
		sagaCtx = NewSagaContext()
	}

	slog.Info("Saga started", "saga_name", s.Name)

	var completedSteps []int

	// Execute forward steps
	for i, step := range s.Steps {
		slog.Info("Executing saga step",
			"saga_name", s.Name,
			"step_name", step.Name,
			"step_index", i,
		)

		if err := step.Action(ctx, sagaCtx); err != nil {
			slog.Error("Saga step failed",
				"saga_name", s.Name,
				"step_name", step.Name,
				"step_index", i,
				"error", err,
			)

			sagaCtx.RecordError(err)

			// Call custom error handler if set
			if s.OnError != nil {
				if handlerErr := s.OnError(ctx, sagaCtx, i, err); handlerErr != nil {
					slog.Error("Saga error handler failed",
						"saga_name", s.Name,
						"error", handlerErr,
					)
				}
			}

			// Compensate completed steps in reverse order
			if compErr := s.compensate(ctx, sagaCtx, completedSteps); compErr != nil {
				slog.Error("Saga compensation failed",
					"saga_name", s.Name,
					"error", compErr,
				)
				return fmt.Errorf("saga %s: step %d failed (%w), compensation also failed: %w",
					s.Name, i, err, compErr)
			}

			return fmt.Errorf("saga %s: step %d (%s) failed: %w", s.Name, i, step.Name, err)
		}

		completedSteps = append(completedSteps, i)
		sagaCtx.RecordStep(step.Name)

		slog.Info("Saga step completed",
			"saga_name", s.Name,
			"step_name", step.Name,
			"step_index", i,
		)
	}

	slog.Info("Saga completed successfully",
		"saga_name", s.Name,
		"steps_completed", len(completedSteps),
		"duration", time.Since(sagaCtx.StartTime),
	)

	return nil
}

// compensate executes compensation steps in reverse order.
func (s *Saga) compensate(ctx context.Context, sagaCtx *SagaContext, completedSteps []int) error {
	if len(completedSteps) == 0 {
		return nil
	}

	slog.Info("Starting saga compensation",
		"saga_name", s.Name,
		"steps_to_compensate", len(completedSteps),
	)

	// Reverse order
	for i := len(completedSteps) - 1; i >= 0; i-- {
		stepIndex := completedSteps[i]
		step := s.Steps[stepIndex]

		if step.Compensate == nil {
			slog.Warn("No compensation defined for step",
				"saga_name", s.Name,
				"step_name", step.Name,
				"step_index", stepIndex,
			)
			continue
		}

		slog.Info("Compensating saga step",
			"saga_name", s.Name,
			"step_name", step.Name,
			"step_index", stepIndex,
		)

		if err := step.Compensate(ctx, sagaCtx); err != nil {
			slog.Error("Compensation step failed",
				"saga_name", s.Name,
				"step_name", step.Name,
				"step_index", stepIndex,
				"error", err,
			)
			// Continue with other compensations even if one fails
			sagaCtx.RecordError(fmt.Errorf("compensation failed for step %d (%s): %w", stepIndex, step.Name, err))
		}
	}

	if len(sagaCtx.Errors) > 0 {
		return fmt.Errorf("compensation completed with %d errors", len(sagaCtx.Errors))
	}

	return nil
}
