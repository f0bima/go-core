package bff

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ClientType represents different client types.
type ClientType string

const (
	ClientWeb    ClientType = "web"
	ClientMobile ClientType = "mobile"
	ClientAdmin  ClientType = "admin"
)

// BFFConfig holds configuration for a BFF instance.
type BFFConfig struct {
	ClientType  ClientType
	BaseURL     string
	Timeout     time.Duration
	Middlewares []gin.HandlerFunc
	Routes      func(router *gin.Engine, adapter *Adapter)
}

// BFF represents a Backend-for-Frontend server.
type BFF struct {
	config  *BFFConfig
	engine  *gin.Engine
	adapter *Adapter
}

// New creates a new BFF instance.
func New(config *BFFConfig) *BFF {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())

	// Add custom middlewares
	for _, mw := range config.Middlewares {
		engine.Use(mw)
	}

	adapter := NewAdapter(config.BaseURL, config.Timeout)

	bff := &BFF{
		config:  config,
		engine:  engine,
		adapter: adapter,
	}

	// Setup routes
	if config.Routes != nil {
		config.Routes(engine, adapter)
	}

	return bff
}

// Engine returns the underlying Gin engine.
func (b *BFF) Engine() *gin.Engine {
	return b.engine
}

// Adapter returns the HTTP adapter for backend service calls.
func (b *BFF) Adapter() *Adapter {
	return b.adapter
}

// Start starts the BFF server.
func (b *BFF) Start(addr string) error {
	if addr == "" {
		addr = ":8080"
	}

	fmt.Printf("BFF [%s] starting on %s\n", b.config.ClientType, addr)
	return b.engine.Run(addr)
}

// Adapter provides HTTP client methods for backend service communication.
type Adapter struct {
	baseURL string
	client  *http.Client
}

// NewAdapter creates a new adapter for backend service calls.
func NewAdapter(baseURL string, timeout time.Duration) *Adapter {
	return &Adapter{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Get performs a GET request to the backend service.
func (a *Adapter) Get(ctx context.Context, path string, headers map[string]string) (*http.Response, error) {
	return a.doRequest(ctx, http.MethodGet, path, nil, headers)
}

// Post performs a POST request to the backend service.
func (a *Adapter) Post(ctx context.Context, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	return a.doRequest(ctx, http.MethodPost, path, body, headers)
}

// Put performs a PUT request to the backend service.
func (a *Adapter) Put(ctx context.Context, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	return a.doRequest(ctx, http.MethodPut, path, body, headers)
}

// Delete performs a DELETE request to the backend service.
func (a *Adapter) Delete(ctx context.Context, path string, headers map[string]string) (*http.Response, error) {
	return a.doRequest(ctx, http.MethodDelete, path, nil, headers)
}

// doRequest performs an HTTP request to the backend service.
func (a *Adapter) doRequest(ctx context.Context, method, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, a.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	return resp, nil
}

// ComposeResponse aggregates multiple backend responses into a single response.
func (a *Adapter) ComposeResponse(ctx context.Context, requests []BackendRequest) (map[string]interface{}, error) {
	results := make(map[string]interface{})

	for _, req := range requests {
		resp, err := a.doRequest(ctx, req.Method, req.Path, nil, req.Headers)
		if err != nil {
			if req.Required {
				return nil, fmt.Errorf("required request failed [%s]: %w", req.Name, err)
			}
			// Optional request failed - skip
			continue
		}

		// Decode response (simplified - in real implementation, parse JSON)
		results[req.Name] = resp
	}

	return results, nil
}

// BackendRequest represents a request to a backend service.
type BackendRequest struct {
	Name     string
	Method   string
	Path     string
	Headers  map[string]string
	Required bool
}
