package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/anto1290/qlxion-monorepo/api-gateway/internal/config"
	"github.com/rs/zerolog/log"
)

// ReverseProxy handles forwarding requests to backend services
type ReverseProxy struct {
	services map[string]*ServiceProxy
	client   *http.Client
	cfg      config.GatewayConfig
}

// ServiceProxy represents a proxy to a backend service
type ServiceProxy struct {
	Name    string
	BaseURL string
	Client  *http.Client
}

// ProxyResponse represents the response from a backend service
type ProxyResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Latency    time.Duration
	Error      error
}

// NewReverseProxy creates a new reverse proxy
func NewReverseProxy(cfg config.GatewayConfig, services []config.Service) *ReverseProxy {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:    cfg.MaxIdleConns,
			MaxConnsPerHost: cfg.MaxConnsPerHost,
			IdleConnTimeout: 90 * time.Second,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2: true,
		},
		Timeout: cfg.RequestTimeout,
	}

	svcMap := make(map[string]*ServiceProxy)
	for _, svc := range services {
		protocol := svc.Protocol
		if protocol == "" {
			protocol = "http"
		}

		baseURL := fmt.Sprintf("%s://%s:%d", protocol, svc.Host, svc.Port)
		svcMap[svc.Name] = &ServiceProxy{
			Name:    svc.Name,
			BaseURL: baseURL,
			Client:  client,
		}

		log.Info().
			Str("service", svc.Name).
			Str("url", baseURL).
			Msg("Service registered")
	}

	return &ReverseProxy{
		services: svcMap,
		client:   client,
		cfg:      cfg,
	}
}

// Forward forwards a request to a backend service
func (rp *ReverseProxy) Forward(ctx context.Context, serviceName string, backendPath string, r *http.Request) (*ProxyResponse, error) {
	svc, ok := rp.services[serviceName]
	if !ok {
		return nil, fmt.Errorf("service not found: %s", serviceName)
	}

	start := time.Now()

	// Build backend URL
	backendURL, err := url.Parse(svc.BaseURL + backendPath)
	if err != nil {
		return nil, fmt.Errorf("invalid backend URL: %w", err)
	}

	// Copy query parameters
	if r.URL.RawQuery != "" {
		backendURL.RawQuery = r.URL.RawQuery
	}

	// Read request body
	var body []byte
	if r.Body != nil {
		body, err = io.ReadAll(r.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		defer r.Body.Close()
	}

	// Create new request
	req, err := http.NewRequestWithContext(ctx, r.Method, backendURL.String(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Set forwarded headers
	req.Header.Set("X-Forwarded-For", r.RemoteAddr)
	req.Header.Set("X-Forwarded-Host", r.Host)
	req.Header.Set("X-Forwarded-Proto", r.URL.Scheme)
	if r.URL.Scheme == "" {
		req.Header.Set("X-Forwarded-Proto", "http")
	}

	// Execute request
	resp, err := svc.Client.Do(req)
	if err != nil {
		return &ProxyResponse{
			Latency: time.Since(start),
			Error:   err,
		}, err
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, int64(rp.cfg.ResponseBufferSize)))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	latency := time.Since(start)

	log.Debug().
		Str("service", serviceName).
		Str("method", r.Method).
		Str("path", backendPath).
		Int("status", resp.StatusCode).
		Dur("latency", latency).
		Msg("Request forwarded")

	return &ProxyResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       respBody,
		Latency:    latency,
	}, nil
}

// ForwardMultiple forwards requests to multiple services concurrently
func (rp *ReverseProxy) ForwardMultiple(ctx context.Context, requests []ForwardRequest) map[string]*ProxyResponse {
	results := make(map[string]*ProxyResponse)
	type result struct {
		key      string
		response *ProxyResponse
	}

	resultChan := make(chan result, len(requests))

	for _, req := range requests {
		go func(fr ForwardRequest) {
			resp, err := rp.Forward(ctx, fr.Service, fr.Path, fr.Request)
			if err != nil {
				resp = &ProxyResponse{Error: err}
			}
			resultChan <- result{key: fr.Key, response: resp}
		}(req)
	}

	for i := 0; i < len(requests); i++ {
		res := <-resultChan
		results[res.key] = res.response
	}

	return results
}

// ForwardRequest represents a request to forward
type ForwardRequest struct {
	Key     string
	Service string
	Path    string
	Request *http.Request
}

// HealthCheck checks if a service is healthy
func (rp *ReverseProxy) HealthCheck(serviceName string) error {
	svc, ok := rp.services[serviceName]
	if !ok {
		return fmt.Errorf("service not found: %s", serviceName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, svc.BaseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := svc.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

// GetServiceNames returns all registered service names
func (rp *ReverseProxy) GetServiceNames() []string {
	names := make([]string, 0, len(rp.services))
	for name := range rp.services {
		names = append(names, name)
	}
	return names
}

// JSONResponse sends a JSON response from proxy response
func JSONResponse(w http.ResponseWriter, pr *ProxyResponse) {
	for key, values := range pr.Headers {
		if key != "Content-Length" && key != "Content-Encoding" {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
	}
	w.WriteHeader(pr.StatusCode)
	w.Write(pr.Body)
}
