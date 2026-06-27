package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds the gateway configuration
type Config struct {
	Server   ServerConfig   `yaml:"server" json:"server"`
	Gateway  GatewayConfig  `yaml:"gateway" json:"gateway"`
	Services []Service      `yaml:"services" json:"services"`
	JWT      JWTConfig      `yaml:"jwt" json:"jwt"`
	Redis    RedisConfig    `yaml:"redis" json:"redis"`
	RateLimit RateLimitConfig `yaml:"rate_limit" json:"rate_limit"`
	CORS     CORSConfig     `yaml:"cors" json:"cors"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host         string        `yaml:"host" json:"host"`
	Port         int           `yaml:"port" json:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" json:"idle_timeout"`
}

// GatewayConfig holds gateway-specific configuration
type GatewayConfig struct {
	Name               string        `yaml:"name" json:"name"`
	Debug              bool          `yaml:"debug" json:"debug"`
	EnableAggregator   bool          `yaml:"enable_aggregator" json:"enable_aggregator"`
	EnableAuth         bool          `yaml:"enable_auth" json:"enable_auth"`
	RequestTimeout     time.Duration `yaml:"request_timeout" json:"request_timeout"`
	MaxIdleConns       int           `yaml:"max_idle_conns" json:"max_idle_conns"`
	MaxConnsPerHost    int           `yaml:"max_conns_per_host" json:"max_conns_per_host"`
	ResponseBufferSize int           `yaml:"response_buffer_size" json:"response_buffer_size"`
}

// Service represents a backend service configuration
type Service struct {
	Name        string            `yaml:"name" json:"name"`
	Host        string            `yaml:"host" json:"host"`
	Port        int               `yaml:"port" json:"port"`
	Protocol    string            `yaml:"protocol" json:"protocol"`
	HealthCheck string            `yaml:"health_check" json:"health_check"`
	Endpoints   []Endpoint        `yaml:"endpoints" json:"endpoints"`
	Headers     map[string]string `yaml:"headers" json:"headers"`
}

// Endpoint represents a service endpoint configuration
type Endpoint struct {
	Path          string            `yaml:"path" json:"path"`
	Method        string            `yaml:"method" json:"method"`
	Backend       string            `yaml:"backend" json:"backend"`
	BackendPath   string            `yaml:"backend_path" json:"backend_path"`
	BackendMethod string            `yaml:"backend_method" json:"backend_method"`
	RequiresAuth  bool              `yaml:"requires_auth" json:"requires_auth"`
	RequiredRoles []string          `yaml:"required_roles" json:"required_roles"`
	RateLimit     *RateLimitRule    `yaml:"rate_limit,omitempty" json:"rate_limit,omitempty"`
	Headers       map[string]string `yaml:"headers" json:"headers"`
	QueryParams   map[string]string `yaml:"query_params" json:"query_params"`
}

// RateLimitRule defines rate limiting for a specific endpoint
type RateLimitRule struct {
	RequestsPerSecond int           `yaml:"requests_per_second" json:"requests_per_second"`
	BurstSize         int           `yaml:"burst_size" json:"burst_size"`
	Window            time.Duration `yaml:"window" json:"window"`
}

// JWTConfig holds JWT validation configuration
type JWTConfig struct {
	Secret            string        `yaml:"secret" json:"secret"`
	PublicKeyPath     string        `yaml:"public_key_path" json:"public_key_path"`
	TokenHeader       string        `yaml:"token_header" json:"token_header"`
	TokenPrefix       string        `yaml:"token_prefix" json:"token_prefix"`
	AllowedAlgorithms []string      `yaml:"allowed_algorithms" json:"allowed_algorithms"`
	ClaimsHeaders     ClaimsHeaders `yaml:"claims_headers" json:"claims_headers"`
}

// ClaimsHeaders maps JWT claims to request headers
type ClaimsHeaders struct {
	UserID   string `yaml:"user_id" json:"user_id"`
	TenantID string `yaml:"tenant_id" json:"tenant_id"`
	Email    string `yaml:"email" json:"email"`
	Roles    string `yaml:"roles" json:"roles"`
}

// RedisConfig holds Redis configuration for rate limiting
type RedisConfig struct {
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	Password string `yaml:"password" json:"password"`
	DB       int    `yaml:"db" json:"db"`
}

// RateLimitConfig holds global rate limiting configuration
type RateLimitConfig struct {
	Enabled           bool          `yaml:"enabled" json:"enabled"`
	DefaultRPS        int           `yaml:"default_rps" json:"default_rps"`
	DefaultBurstSize  int           `yaml:"default_burst_size" json:"default_burst_size"`
	DefaultWindow     time.Duration `yaml:"default_window" json:"default_window"`
	RedisKeyPrefix    string        `yaml:"redis_key_prefix" json:"redis_key_prefix"`
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	Enabled          bool     `yaml:"enabled" json:"enabled"`
	AllowedOrigins   []string `yaml:"allowed_origins" json:"allowed_origins"`
	AllowedMethods   []string `yaml:"allowed_methods" json:"allowed_methods"`
	AllowedHeaders   []string `yaml:"allowed_headers" json:"allowed_headers"`
	ExposedHeaders   []string `yaml:"exposed_headers" json:"exposed_headers"`
	AllowCredentials bool     `yaml:"allow_credentials" json:"allow_credentials"`
	MaxAge           int      `yaml:"max_age" json:"max_age"`
}

// Load loads configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	setDefaults(&cfg)

	return &cfg, nil
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() *Config {
	cfg := &Config{}
	
	cfg.Server.Host = getEnv("GATEWAY_HOST", "0.0.0.0")
	cfg.Server.Port = getEnvInt("GATEWAY_PORT", 8000)
	cfg.Server.ReadTimeout = getEnvDuration("GATEWAY_READ_TIMEOUT", 10*time.Second)
	cfg.Server.WriteTimeout = getEnvDuration("GATEWAY_WRITE_TIMEOUT", 10*time.Second)
	cfg.Server.IdleTimeout = getEnvDuration("GATEWAY_IDLE_TIMEOUT", 60*time.Second)

	cfg.Gateway.Name = getEnv("GATEWAY_NAME", "qlxion-gateway")
	cfg.Gateway.Debug = getEnvBool("GATEWAY_DEBUG", false)
	cfg.Gateway.EnableAuth = getEnvBool("GATEWAY_ENABLE_AUTH", true)
	cfg.Gateway.RequestTimeout = getEnvDuration("GATEWAY_REQUEST_TIMEOUT", 30*time.Second)

	cfg.JWT.Secret = getEnv("JWT_SECRET", "your-secret-key-change-in-production")
	cfg.JWT.TokenHeader = getEnv("JWT_HEADER", "Authorization")
	cfg.JWT.TokenPrefix = getEnv("JWT_PREFIX", "Bearer ")

	cfg.Redis.Host = getEnv("REDIS_HOST", "localhost")
	cfg.Redis.Port = getEnvInt("REDIS_PORT", 6379)

	cfg.RateLimit.Enabled = getEnvBool("RATE_LIMIT_ENABLED", true)
	cfg.RateLimit.DefaultRPS = getEnvInt("RATE_LIMIT_RPS", 100)

	cfg.CORS.Enabled = getEnvBool("CORS_ENABLED", true)
	cfg.CORS.AllowedOrigins = []string{"*"}
	cfg.CORS.AllowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	cfg.CORS.AllowedHeaders = []string{"*"}

	return cfg
}

func setDefaults(cfg *Config) {
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8000
	}
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = 10 * time.Second
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = 10 * time.Second
	}
	if cfg.Gateway.RequestTimeout == 0 {
		cfg.Gateway.RequestTimeout = 30 * time.Second
	}
	if cfg.JWT.TokenHeader == "" {
		cfg.JWT.TokenHeader = "Authorization"
	}
	if cfg.JWT.TokenPrefix == "" {
		cfg.JWT.TokenPrefix = "Bearer "
	}
	if cfg.RateLimit.DefaultRPS == 0 {
		cfg.RateLimit.DefaultRPS = 100
	}
	if cfg.RateLimit.DefaultBurstSize == 0 {
		cfg.RateLimit.DefaultBurstSize = 150
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	// Simplified - in production use strconv.Atoi
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	// Simplified - in production use strconv.ParseBool
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	// Simplified - in production use time.ParseDuration
	return defaultValue
}
