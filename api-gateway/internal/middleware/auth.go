package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/anto1290/qlxion-monorepo/api-gateway/internal/config"
	"github.com/anto1290/qlxion-monorepo/pkg/auth"
	"github.com/anto1290/qlxion-monorepo/pkg/response"
)

// Auth middleware validates JWT tokens
func Auth(jwtConfig config.JWTConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from header
			authHeader := r.Header.Get(jwtConfig.TokenHeader)
			if authHeader == "" {
				resp := response.Error(response.New(response.ErrUnauthorized, "Missing authorization header"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(resp)
				return
			}

			// Remove prefix
			tokenString := authHeader
			if jwtConfig.TokenPrefix != "" {
				tokenString = strings.TrimPrefix(authHeader, jwtConfig.TokenPrefix)
			}
			tokenString = strings.TrimSpace(tokenString)

			// Validate token
			claims, err := auth.ValidateToken(tokenString, jwtConfig.Secret)
			if err != nil {
				resp := response.Error(response.New(response.ErrTokenInvalid, "Invalid or expired token").WithDetail(err.Error()))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(resp)
				return
			}

			// Add claims to request context
			ctx := auth.ContextWithClaims(r.Context(), claims)

			// Set claims headers for downstream services
			if jwtConfig.ClaimsHeaders.UserID != "" && claims.UserID != [16]byte{} {
				r.Header.Set(jwtConfig.ClaimsHeaders.UserID, claims.UserID.String())
			}
			if jwtConfig.ClaimsHeaders.TenantID != "" && claims.TenantID != [16]byte{} {
				r.Header.Set(jwtConfig.ClaimsHeaders.TenantID, claims.TenantID.String())
			}
			if jwtConfig.ClaimsHeaders.Email != "" {
				r.Header.Set(jwtConfig.ClaimsHeaders.Email, claims.Email)
			}
			if jwtConfig.ClaimsHeaders.Roles != "" && len(claims.Roles) > 0 {
				r.Header.Set(jwtConfig.ClaimsHeaders.Roles, strings.Join(claims.Roles, ","))
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth is like Auth but doesn't require authentication
func OptionalAuth(jwtConfig config.JWTConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get(jwtConfig.TokenHeader)
			if authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			tokenString := authHeader
			if jwtConfig.TokenPrefix != "" {
				tokenString = strings.TrimPrefix(authHeader, jwtConfig.TokenPrefix)
			}
			tokenString = strings.TrimSpace(tokenString)

			claims, err := auth.ValidateToken(tokenString, jwtConfig.Secret)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := auth.ContextWithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole checks if the authenticated user has the required role
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.ExtractClaimsFromContext(r.Context())
			if !ok {
				resp := response.Error(response.New(response.ErrUnauthorized, "Authentication required"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(resp)
				return
			}

			hasRole := false
			for _, role := range roles {
				if claims.HasRole(role) {
					hasRole = true
					break
				}
			}

			if !hasRole {
				resp := response.Error(response.New(response.ErrForbidden, "Insufficient permissions"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(resp)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ContextWithValue adds a value to context (helper function)
func ContextWithValue(ctx context.Context, key, value string) context.Context {
	return context.WithValue(ctx, key, value)
}
