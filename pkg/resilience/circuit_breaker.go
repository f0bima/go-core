package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

// CircuitState represents the state of circuit breaker.
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig holds configuration for circuit breaker.
type CircuitBreakerConfig struct {
	FailureThreshold int           // Number of failures before opening circuit
	RecoveryTimeout  time.Duration // Time to wait before transitioning to half-open
	SuccessThreshold int           // Successes needed in half-open to close circuit
}

// DefaultCircuitBreakerConfig returns sensible defaults.
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		FailureThreshold: 5,
		RecoveryTimeout:  30 * time.Second,
		SuccessThreshold: 3,
	}
}

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	mu               sync.Mutex
	state            CircuitState
	failures         int
	successes        int
	lastFailureTime  time.Time
	config           *CircuitBreakerConfig
	onStateChange    func(oldState, newState CircuitState)
}

// ErrCircuitOpen is returned when circuit breaker is open.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		state:  StateClosed,
		config: config,
	}
}

// WithStateChangeCallback sets a callback for state transitions.
func (cb *CircuitBreaker) WithStateChangeCallback(fn func(oldState, newState CircuitState)) *CircuitBreaker {
	cb.onStateChange = fn
	return cb
}

// Execute runs the given function through the circuit breaker.
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	cb.mu.Lock()

	// Check if request should be allowed
	if !cb.allowRequest() {
		cb.mu.Unlock()
		return ErrCircuitOpen
	}

	cb.mu.Unlock()

	// Execute the function
	err := fn(ctx)

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.recordFailure()
		return err
	}

	cb.recordSuccess()
	return nil
}

// GetState returns the current state of the circuit breaker.
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// allowRequest checks if a request should be allowed.
func (cb *CircuitBreaker) allowRequest() bool {
	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if recovery timeout has elapsed
		if time.Since(cb.lastFailureTime) > cb.config.RecoveryTimeout {
			cb.transitionTo(StateHalfOpen)
			return true
		}
		return false
	case StateHalfOpen:
		return true
	}
	return false
}

// recordFailure records a failure and potentially opens the circuit.
func (cb *CircuitBreaker) recordFailure() {
	cb.failures++
	cb.lastFailureTime = time.Now()
	cb.successes = 0

	if cb.state == StateHalfOpen {
		cb.transitionTo(StateOpen)
	} else if cb.failures >= cb.config.FailureThreshold {
		cb.transitionTo(StateOpen)
	}
}

// recordSuccess records a success and potentially closes the circuit.
func (cb *CircuitBreaker) recordSuccess() {
	if cb.state == StateHalfOpen {
		cb.successes++
		if cb.successes >= cb.config.SuccessThreshold {
			cb.transitionTo(StateClosed)
		}
	} else if cb.state == StateClosed {
		// Reset failure count on success
		cb.failures = 0
	}
}

// transitionTo changes the circuit breaker state.
func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	oldState := cb.state
	cb.state = newState

	// Reset counters on state transition
	if newState == StateClosed {
		cb.failures = 0
		cb.successes = 0
	} else if newState == StateHalfOpen {
		cb.successes = 0
	}

	if cb.onStateChange != nil {
		cb.onStateChange(oldState, newState)
	}
}

// Reset manually resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.transitionTo(StateClosed)
}
