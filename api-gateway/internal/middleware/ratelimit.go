package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/qlxion/qlxion-monorepo/api-gateway/internal/config"
	"github.com/qlxion/qlxion-monorepo/pkg/response"
	"context"
	"encoding/json"
)

// RateLimiter handles rate limiting using sliding window algorithm
type RateLimiter struct {
	redis      *redis.Client
	config     config.RateLimitConfig
	localCache map[string]*localCounter
	mu         sync.RWMutex
}

type localCounter struct {
	count  int
	window time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(redis *redis.Client, cfg config.RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		redis:      redis,
		config:     cfg,
		localCache: make(map[string]*localCounter),
	}
	
	// Start cleanup goroutine
	go rl.cleanupLoop()
	
	return rl
}

// Middleware returns the rate limiting middleware
func (rl *RateLimiter) Middleware(rps int, burst int, window time.Duration) func(http.Handler) http.Handler {
	if !rl.config.Enabled {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	// Use defaults if not specified
	if rps == 0 {
		rps = rl.config.DefaultRPS
	}
	if burst == 0 {
		burst = rl.config.DefaultBurstSize
	}
	if window == 0 {
		window = rl.config.DefaultWindow
		if window == 0 {
			window = time.Minute
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := rl.generateKey(r)

			allowed, retryAfter := rl.isAllowed(r.Context(), key, rps, burst, window)
			if !allowed {
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rps))
				w.Header().Set("X-RateLimit-Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
				
				resp := response.Fail(nil, "Rate limit exceeded. Please try again later.")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(resp)
				return
			}

			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rps))
			next.ServeHTTP(w, r)
		})
	}
}

// isAllowed checks if the request is allowed
func (rl *RateLimiter) isAllowed(ctx context.Context, key string, rps, burst int, window time.Duration) (bool, time.Duration) {
	// Try Redis first (distributed rate limiting)
	if rl.redis != nil {
		return rl.isAllowedRedis(ctx, key, rps, burst, window)
	}

	// Fallback to local rate limiting
	return rl.isAllowedLocal(key, rps, burst, window)
}

// isAllowedRedis uses Redis for distributed rate limiting
func (rl *RateLimiter) isAllowedRedis(ctx context.Context, key string, rps, burst int, window time.Duration) (bool, time.Duration) {
	redisKey := fmt.Sprintf("%s:%s", rl.config.RedisKeyPrefix, key)
	now := time.Now().Unix()
	windowStart := now - int64(window.Seconds())

	// Remove old entries and count current window
	pipe := rl.redis.Pipeline()
	pipe.ZRemRangeByScore(ctx, redisKey, "0", fmt.Sprintf("%d", windowStart))
	pipe.ZCard(ctx, redisKey)
	
	results, err := pipe.Exec(ctx)
	if err != nil {
		// On Redis error, allow the request (fail open) or use local fallback
		return rl.isAllowedLocal(key, rps, burst, window)
	}

	currentCount := results[1].(*redis.IntCmd).Val()
	
	if int(currentCount) >= burst {
		// Get oldest entry for retry-after calculation
		oldest, _ := rl.redis.ZRangeWithScores(ctx, redisKey, 0, 0).Result()
		var retryAfter time.Duration
		if len(oldest) > 0 {
			oldestTimestamp := time.Unix(int64(oldest[0].Score), 0)
			retryAfter = window - time.Since(oldestTimestamp)
			if retryAfter < 0 {
				retryAfter = 0
			}
		}
		return false, retryAfter
	}

	// Add current request
	member := fmt.Sprintf("%d:%s", now, key)
	rl.redis.ZAdd(ctx, redisKey, redis.Z{
		Score:  float64(now),
		Member: member,
	})
	rl.redis.Expire(ctx, redisKey, window)

	return true, 0
}

// isAllowedLocal uses in-memory map for local rate limiting
func (rl *RateLimiter) isAllowedLocal(key string, rps, burst int, window time.Duration) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	counter, exists := rl.localCache[key]

	if !exists || now.Sub(counter.window) > window {
		// New window
		rl.localCache[key] = &localCounter{
			count:  1,
			window: now,
		}
		return true, 0
	}

	if counter.count >= burst {
		retryAfter := window - now.Sub(counter.window)
		if retryAfter < 0 {
			retryAfter = 0
		}
		return false, retryAfter
	}

	counter.count++
	return true, 0
}

// generateKey creates a unique key for rate limiting
func (rl *RateLimiter) generateKey(r *http.Request) string {
	// Use client IP + path as key
	clientIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		clientIP = forwarded
	}
	return fmt.Sprintf("%s:%s:%s", clientIP, r.Method, r.URL.Path)
}

// cleanupLoop periodically cleans up expired local cache entries
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, counter := range rl.localCache {
			if now.Sub(counter.window) > 2*time.Minute {
				delete(rl.localCache, key)
			}
		}
		rl.mu.Unlock()
	}
}

// GlobalRateLimit returns a global rate limit middleware
func (rl *RateLimiter) GlobalRateLimit() func(http.Handler) http.Handler {
	return rl.Middleware(rl.config.DefaultRPS, rl.config.DefaultBurstSize, time.Minute)
}
