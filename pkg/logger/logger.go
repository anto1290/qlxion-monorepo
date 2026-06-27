package logger

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Config holds logger configuration
type Config struct {
	Level      string
	Format     string // json or console
	TimeFormat string
}

// DefaultConfig returns default logger configuration
func DefaultConfig() Config {
	return Config{
		Level:      "info",
		Format:     "json",
		TimeFormat: time.RFC3339,
	}
}

// Init initializes the global logger
func Init(cfg Config) {
	level, err := zerolog.ParseLevel(strings.ToLower(cfg.Level))
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	zerolog.TimeFieldFormat = cfg.TimeFormat

	if strings.ToLower(cfg.Format) == "console" {
		log.Logger = zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: cfg.TimeFormat,
			NoColor:    false,
		}).With().Timestamp().Caller().Logger()
	} else {
		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()
	}
}

// Logger wraps zerolog.Logger with additional context methods
type Logger struct {
	zerolog.Logger
}

// New creates a new logger instance
func New(service string) *Logger {
	return &Logger{
		Logger: log.Logger.With().Str("service", service).Logger(),
	}
}

// WithContext adds context fields to logger
func (l *Logger) WithContext(ctx context.Context) *Logger {
	logger := l.Logger.With()
	
	if requestID, ok := ctx.Value("request_id").(string); ok {
		logger = logger.Str("request_id", requestID)
	}
	if userID, ok := ctx.Value("user_id").(string); ok {
		logger = logger.Str("user_id", userID)
	}
	if tenantID, ok := ctx.Value("tenant_id").(string); ok {
		logger = logger.Str("tenant_id", tenantID)
	}

	return &Logger{Logger: logger.Logger()}
}

// WithField adds a single field to logger
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{Logger: l.Logger.With().Interface(key, value).Logger()}
}

// WithError adds error to logger
func (l *Logger) WithError(err error) *Logger {
	return &Logger{Logger: l.Logger.With().Err(err).Logger()}
}

// RequestLogger middleware logging helper
type RequestLog struct {
	Method     string        `json:"method"`
	Path       string        `json:"path"`
	Status     int           `json:"status"`
	Duration   time.Duration `json:"duration"`
	RequestID  string        `json:"request_id"`
	UserID     string        `json:"user_id,omitempty"`
	TenantID   string        `json:"tenant_id,omitempty"`
	ClientIP   string        `json:"client_ip"`
	UserAgent  string        `json:"user_agent"`
	Error      string        `json:"error,omitempty"`
}

// LogRequest logs HTTP request details
func (l *Logger) LogRequest(req RequestLog) {
	event := l.Logger.Info().
		Str("method", req.Method).
		Str("path", req.Path).
		Int("status", req.Status).
		Dur("duration", req.Duration).
		Str("request_id", req.RequestID).
		Str("client_ip", req.ClientIP).
		Str("user_agent", req.UserAgent)

	if req.UserID != "" {
		event = event.Str("user_id", req.UserID)
	}
	if req.TenantID != "" {
		event = event.Str("tenant_id", req.TenantID)
	}
	if req.Error != "" {
		event = event.Str("error", req.Error)
	}

	msg := "HTTP Request"
	if req.Status >= 500 {
		event.Msgf("%s - Server Error", msg)
	} else if req.Status >= 400 {
		event.Msgf("%s - Client Error", msg)
	} else {
		event.Msg(msg)
	}
}
