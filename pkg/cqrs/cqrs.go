package cqrs

import (
	"context"
	"errors"
	"fmt"
)

// Command represents a write operation that changes state.
type Command interface {
	Validate() error
}

// Query represents a read operation.
type Query interface {
	Validate() error
}

// CommandHandler handles a specific command type.
type CommandHandler[C Command] interface {
	Handle(ctx context.Context, cmd C) error
}

// QueryHandler handles a specific query type and returns a result.
type QueryHandler[Q Query, R any] interface {
	Handle(ctx context.Context, query Q) (R, error)
}

// CommandBus dispatches commands to appropriate handlers.
type CommandBus struct {
	handlers map[string]interface{}
}

// NewCommandBus creates a new command bus.
func NewCommandBus() *CommandBus {
	return &CommandBus{
		handlers: make(map[string]interface{}),
	}
}

// Register registers a command handler.
func (b *CommandBus) Register(commandType string, handler interface{}) {
	b.handlers[commandType] = handler
}

// Dispatch dispatches a command to its handler.
func (b *CommandBus) Dispatch(ctx context.Context, cmd Command) error {
	handler, exists := b.handlers[getTypeName(cmd)]
	if !exists {
		return ErrHandlerNotFound
	}

	// Type assert and call
	switch h := handler.(type) {
	case CommandHandler[Command]:
		return h.Handle(ctx, cmd)
	default:
		return ErrInvalidHandler
	}
}

// QueryBus dispatches queries to appropriate handlers.
type QueryBus struct {
	handlers map[string]interface{}
}

// NewQueryBus creates a new query bus.
func NewQueryBus() *QueryBus {
	return &QueryBus{
		handlers: make(map[string]interface{}),
	}
}

// Register registers a query handler.
func (b *QueryBus) Register(queryType string, handler interface{}) {
	b.handlers[queryType] = handler
}

// Dispatch dispatches a query to its handler.
func (b *QueryBus) Dispatch(ctx context.Context, query Query) (interface{}, error) {
	handler, exists := b.handlers[getTypeName(query)]
	if !exists {
		return nil, ErrHandlerNotFound
	}

	// Type assert and call
	switch h := handler.(type) {
	case QueryHandler[Query, any]:
		return h.Handle(ctx, query)
	default:
		return nil, ErrInvalidHandler
	}
}

// DispatchTyped dispatches a query with typed result.
func DispatchTyped[R any](bus *QueryBus, ctx context.Context, query Query) (R, error) {
	result, err := bus.Dispatch(ctx, query)
	if err != nil {
		var zero R
		return zero, err
	}

	if r, ok := result.(R); ok {
		return r, nil
	}

	var zero R
	return zero, ErrInvalidResultType
}

// getTypeName returns the type name of a command or query.
func getTypeName(obj interface{}) string {
	switch v := obj.(type) {
	case Command:
		return fmt.Sprintf("%T", v)
	case Query:
		return fmt.Sprintf("%T", v)
	default:
		return "unknown"
	}
}

var (
	// ErrHandlerNotFound is returned when no handler is registered for a command or query.
	ErrHandlerNotFound = errors.New("handler not found")
	// ErrInvalidHandler is returned when the handler type is invalid.
	ErrInvalidHandler = errors.New("invalid handler type")
	// ErrInvalidResultType is returned when the result type doesn't match expected type.
	ErrInvalidResultType = errors.New("invalid result type")
)
