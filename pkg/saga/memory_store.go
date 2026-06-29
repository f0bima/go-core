package saga

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// InMemoryStore provides an in-memory implementation of SagaStore.
// Suitable for testing and development, not for production.
type InMemoryStore struct {
	mu    sync.RWMutex
	states map[uuid.UUID][]byte
}

// NewInMemoryStore creates a new in-memory saga store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		states: make(map[uuid.UUID][]byte),
	}
}

// SaveSagaState saves saga state to memory.
func (s *InMemoryStore) SaveSagaState(ctx context.Context, sagaID uuid.UUID, state []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.states[sagaID] = state
	return nil
}

// GetSagaState retrieves saga state from memory.
func (s *InMemoryStore) GetSagaState(ctx context.Context, sagaID uuid.UUID) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, exists := s.states[sagaID]
	if !exists {
		return nil, fmt.Errorf("saga state not found: %s", sagaID)
	}

	return state, nil
}

// GetIncompleteSagas returns all saga IDs (simplified for in-memory).
func (s *InMemoryStore) GetIncompleteSagas(ctx context.Context) ([]uuid.UUID, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]uuid.UUID, 0, len(s.states))
	for id := range s.states {
		ids = append(ids, id)
	}

	return ids, nil
}

// GetAllStates returns all stored states (for debugging/monitoring).
func (s *InMemoryStore) GetAllStates() map[uuid.UUID][]byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[uuid.UUID][]byte)
	for k, v := range s.states {
		result[k] = v
	}

	return result
}

// Clear removes all stored states.
func (s *InMemoryStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states = make(map[uuid.UUID][]byte)
}
