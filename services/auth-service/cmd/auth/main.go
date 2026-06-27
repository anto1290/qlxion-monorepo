package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anto1290/qlxion-monorepo/pkg/auth"
	"github.com/anto1290/qlxion-monorepo/pkg/database"
	"github.com/anto1290/qlxion-monorepo/pkg/logger"
	"github.com/anto1290/qlxion-monorepo/pkg/response"
	handler "github.com/anto1290/qlxion-monorepo/services/auth-service/internal/delivery/http"
	"github.com/anto1290/qlxion-monorepo/services/auth-service/internal/repository/postgres"
	redisRepo "github.com/anto1290/qlxion-monorepo/services/auth-service/internal/repository/redis"
	"github.com/anto1290/qlxion-monorepo/services/auth-service/internal/usecase"
)

func main() {
	// Initialize logger
	logger.Init(logger.DefaultConfig())
	log := logger.New("auth-service")

	// Load configuration from environment
	cfg := loadConfig()

	if cfg.Debug {
		logger.Init(logger.Config{Level: "debug", Format: "console"})
		log = logger.New("auth-service")
	}

	// Connect to PostgreSQL
	dbCfg := database.PostgresConfig{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		Database: cfg.DBName,
		SSLMode:  cfg.DBSSLMode,
	}

	db, err := database.NewPostgresPool(dbCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer database.Close(db)

	// Connect to Redis
	redisCfg := database.RedisConfig{
		Host:     cfg.RedisHost,
		Port:     cfg.RedisPort,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}
	redisClient := database.NewRedisClient(redisCfg)
	defer database.CloseRedis(redisClient)

	// Initialize repositories
	userRepo := postgres.NewUserRepo(db)
	roleRepo := postgres.NewRoleRepo(db)
	tenantRepo := postgres.NewTenantRepo(db)
	sessionRepo := postgres.NewSessionRepo(db)
	auditRepo := postgres.NewAuditRepo(db)
	cacheRepo := redisRepo.NewCacheRepo(redisClient, "auth")

	// Initialize usecases
	jwtConfig := auth.JWTConfig{
		AccessTokenSecret:  cfg.JWTSecret,
		RefreshTokenSecret: cfg.JWTSecret,
		AccessTokenTTL:     15 * time.Minute,
		RefreshTokenTTL:    7 * 24 * time.Hour,
		Issuer:             "qlxion-auth-service",
	}

	authUC := usecase.NewAuthUsecase(userRepo, roleRepo, tenantRepo, sessionRepo, auditRepo, jwtConfig)
	userUC := usecase.NewUserUsecase(userRepo, roleRepo, tenantRepo, auditRepo)
	roleUC := usecase.NewRoleUsecase(roleRepo, userRepo, auditRepo)
	tenantUC := usecase.NewTenantUsecase(tenantRepo, auditRepo)
	sessionUC := usecase.NewSessionUsecase(sessionRepo, auditRepo)
	auditUC := usecase.NewAuditUsecase(auditRepo)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authUC)
	userHandler := handler.NewUserHandler(userUC)
	roleHandler := handler.NewRoleHandler(roleUC)
	tenantHandler := handler.NewTenantHandler(tenantUC)
	sessionHandler := handler.NewSessionHandler(sessionUC)
	auditHandler := handler.NewAuditHandler(auditUC)

	// Setup router
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		health := map[string]interface{}{
			"status":  "healthy",
			"service": "auth-service",
			"time":    time.Now().UTC(),
		}
		response.JSONSuccess(w, health)
	})

	// Register all handler routes
	authHandler.RegisterRoutes(mux)
	userHandler.RegisterRoutes(mux)
	roleHandler.RegisterRoutes(mux)
	tenantHandler.RegisterRoutes(mux)
	sessionHandler.RegisterRoutes(mux)
	auditHandler.RegisterRoutes(mux)

	// Setup middleware chain
	var handler http.Handler = mux

	// Recovery middleware
	handler = recoveryMiddleware(log)(handler)

	// Request logging
	handler = requestLoggerMiddleware(log)(handler)

	// CORS
	handler = corsMiddleware()(handler)

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info().
			Str("addr", addr).
			Str("database", cfg.DBName).
			Msg("Auth service starting")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	// Wait for interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down auth service...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Auth service stopped")
}

// Config holds service configuration
type Config struct {
	Host          string
	Port          int
	Debug         bool
	DBHost        string
	DBPort        int
	DBUser        string
	DBPassword    string
	DBName        string
	DBSSLMode     string
	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int
	JWTSecret     string
}

func loadConfig() Config {
	return Config{
		Host:          getEnv("AUTH_SERVICE_HOST", "0.0.0.0"),
		Port:          getEnvInt("AUTH_SERVICE_PORT", 8001),
		Debug:         getEnvBool("AUTH_SERVICE_DEBUG", false),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnvInt("DB_PORT", 5432),
		DBUser:        getEnv("DB_USER", "postgres"),
		DBPassword:    getEnv("DB_PASSWORD", "postgres"),
		DBName:        getEnv("DB_NAME", "auth_db"),
		DBSSLMode:     getEnv("DB_SSLMODE", "disable"),
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnvInt("REDIS_PORT", 6379),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),
		JWTSecret:     getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	return defaultValue
}

func recoveryMiddleware(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Error().
						Interface("error", err).
						Str("path", r.URL.Path).
						Msg("Panic recovered")
					response.JSONError(w, response.New(response.ErrInternal, "Internal server error"))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func requestLoggerMiddleware(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			log.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("remote_addr", r.RemoteAddr).
				Dur("duration", time.Since(start)).
				Msg("HTTP Request")
		})
	}
}

func corsMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
			w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
