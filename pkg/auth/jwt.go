package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims represents the JWT claims structure used across services
type Claims struct {
	UserID      uuid.UUID   `json:"user_id"`
	TenantID    uuid.UUID   `json:"tenant_id"`
	Email       string      `json:"email"`
	Roles       []string    `json:"roles"`
	Permissions []string    `json:"permissions"`
	SessionID   uuid.UUID   `json:"session_id"`
	ClientID    string      `json:"client_id,omitempty"`
	jwt.RegisteredClaims
}

// JWTConfig holds configuration for JWT operations
type JWTConfig struct {
	AccessTokenSecret  string
	RefreshTokenSecret string
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
	Issuer             string
}

// DefaultConfig returns a default JWT configuration
func DefaultConfig() JWTConfig {
	return JWTConfig{
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		Issuer:          "qlxion-auth-service",
	}
}

// GenerateAccessToken creates a new JWT access token
func GenerateAccessToken(claims Claims, secret string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims.RegisteredClaims = jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		Issuer:    "qlxion-auth-service",
		Subject:   claims.UserID.String(),
		ID:        uuid.New().String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateRefreshToken creates a new refresh token identifier
func GenerateRefreshToken() (string, error) {
	return uuid.New().String(), nil
}

// ValidateToken validates a JWT token and returns the claims
func ValidateToken(tokenString string, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

// ExtractClaimsFromContext extracts claims from context
func ExtractClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value("claims").(*Claims)
	return claims, ok
}

// ContextWithClaims adds claims to context
func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, "claims", claims)
}

// HasPermission checks if claims contain a specific permission
func (c *Claims) HasPermission(resource, action string) bool {
	required := fmt.Sprintf("%s:%s", resource, action)
	for _, perm := range c.Permissions {
		if perm == required || perm == "*:*" {
			return true
		}
	}
	return false
}

// HasRole checks if claims contain a specific role
func (c *Claims) HasRole(role string) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}
