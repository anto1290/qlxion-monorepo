package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qlxion/qlxion-monorepo/api-gateway/internal/aggregator"
	"github.com/qlxion/qlxion-monorepo/api-gateway/internal/config"
	"github.com/qlxion/qlxion-monorepo/api-gateway/internal/middleware"
	"github.com/qlxion/qlxion-monorepo/api-gateway/internal/proxy"
	"github.com/qlxion/qlxion-monorepo/pkg/logger"
	"github.com/qlxion/qlxion-monorepo/pkg/response"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Initialize logger
	logger.Init(logger.DefaultConfig())
	log := logger.New("api-gateway")

	// Load configuration
	var cfg *config.Config
	configPath := os.Getenv("GATEWAY_CONFIG_PATH")
	if configPath != "" {
		var err error
		cfg, err = config.Load(configPath)
		if err != nil {
			log.Error().Err(err).Msg("Failed to load config file, using env vars")
			cfg = config.LoadFromEnv()
		}
	} else {
		cfg = config.LoadFromEnv()
	}

	if cfg.Gateway.Debug {
		logger.Init(logger.Config{Level: "debug", Format: "console"})
		log = logger.New("api-gateway")
	}

	// Connect to Redis
	var redisClient *redis.Client
	if cfg.RateLimit.Enabled {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
	}

	// Initialize reverse proxy
	rp := proxy.NewReverseProxy(cfg.Gateway, cfg.Services)

	// Initialize aggregator
	agg := aggregator.NewAggregator(rp)

	// Initialize rate limiter
	rateLimiter := middleware.NewRateLimiter(redisClient, cfg.RateLimit)

	// Initialize router
	router := http.NewServeMux()

	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		health := map[string]interface{}{
			"status":    "healthy",
			"service":   cfg.Gateway.Name,
			"timestamp": time.Now().UTC(),
			"version":   "1.0.0",
		}
		response.JSONSuccess(w, health)
	})

	// Gateway info endpoint
	router.HandleFunc("/gateway/info", func(w http.ResponseWriter, r *http.Request) {
		info := map[string]interface{}{
			"name":      cfg.Gateway.Name,
			"version":   "1.0.0",
			"services":  rp.GetServiceNames(),
			"endpoints": getEndpointList(cfg.Services),
		}
		response.JSONSuccess(w, info)
	})

	// Register service endpoints
	for _, svc := range cfg.Services {
		for _, endpoint := range svc.Endpoints {
			route := fmt.Sprintf("%s %s", endpoint.Method, endpoint.Path)
			
			handler := createEndpointHandler(rp, agg, svc.Name, endpoint)
			
			// Apply middleware chain
			var wrappedHandler http.Handler = http.HandlerFunc(handler)
			
			// Apply rate limiting if configured
			if endpoint.RateLimit != nil {
				wrappedHandler = rateLimiter.Middleware(
					endpoint.RateLimit.RequestsPerSecond,
					endpoint.RateLimit.BurstSize,
					endpoint.RateLimit.Window,
				)(wrappedHandler)
			}
			
			// Apply auth if required
			if endpoint.RequiresAuth && cfg.Gateway.EnableAuth {
				wrappedHandler = middleware.Auth(cfg.JWT)(wrappedHandler)
			}
			
			router.Handle(route, wrappedHandler)
		}
	}

	// Create server with middleware chain
	var handler http.Handler = router
	
	// Apply global middleware
	handler = middleware.RequestLogger(log)(handler)
	handler = middleware.CORS(cfg.CORS)(handler)
	handler = rateLimiter.GlobalRateLimit()(handler)
	
	// Recovery middleware
	handler = recoveryMiddleware(log)(handler)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Info().
			Str("addr", addr).
			Str("name", cfg.Gateway.Name).
			Msg("API Gateway starting")
		
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down API Gateway...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	if redisClient != nil {
		redisClient.Close()
	}

	log.Info().Msg("API Gateway stopped")
}

// createEndpointHandler creates an HTTP handler for a service endpoint
func createEndpointHandler(
	rp *proxy.ReverseProxy,
	agg *aggregator.Aggregator,
	serviceName string,
	endpoint config.Endpoint,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Forward request to backend
		pr, err := rp.Forward(r.Context(), serviceName, endpoint.BackendPath, r)
		if err != nil {
			response.JSONError(w, response.New(response.ErrServiceUnavailable, 
				fmt.Sprintf("Service '%s' unavailable", serviceName)).WithError(err))
			return
		}

		// Check if backend returned error status
		if pr.StatusCode >= 500 {
			response.JSONError(w, response.New(response.ErrServiceUnavailable,
				fmt.Sprintf("Service '%s' returned error", serviceName)))
			return
		}

		// Copy response from backend
		proxy.JSONResponse(w, pr)
	}
}

// recoveryMiddleware recovers from panics
func recoveryMiddleware(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Error().
						Interface("error", err).
						Str("path", r.URL.Path).
						Str("method", r.Method).
						Msg("Panic recovered")
					
					response.JSONError(w, response.New(response.ErrInternal, "Internal server error"))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// getEndpointList returns a list of all registered endpoints
func getEndpointList(services []config.Service) []map[string]string {
	var endpoints []map[string]string
	for _, svc := range services {
		for _, ep := range svc.Endpoints {
			endpoints = append(endpoints, map[string]string{
				"path":    ep.Path,
				"method":  ep.Method,
				"service": svc.Name,
			})
		}
	}
	return endpoints
}
