package domain

import (
	"time"

	"github.com/google/uuid"
)

// Session represents a user session
type Session struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	UserID         uuid.UUID  `json:"user_id" db:"user_id"`
	RefreshTokenHash string   `json:"-" db:"refresh_token_hash"`
	AccessTokenID  *string    `json:"access_token_id,omitempty" db:"access_token_id"`
	DeviceName     *string    `json:"device_name,omitempty" db:"device_name"`
	DeviceType     *string    `json:"device_type,omitempty" db:"device_type"` // mobile, desktop, web
	IPAddress      *string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent      string     `json:"user_agent" db:"user_agent"`
	ExpiresAt      time.Time  `json:"expires_at" db:"expires_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
	LastActivityAt *time.Time `json:"last_activity_at,omitempty" db:"last_activity_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`

	// Relations
	User *User `json:"user,omitempty" db:"-"`
}

// TableName returns the table name
func (Session) TableName() string {
	return "sessions"
}

// IsRevoked checks if session is revoked
func (s *Session) IsRevoked() bool {
	return s.RevokedAt != nil || time.Now().After(s.ExpiresAt)
}

// Revoke marks the session as revoked
func (s *Session) Revoke() {
	now := time.Now()
	s.RevokedAt = &now
	s.UpdatedAt = now
}

// UpdateActivity updates the last activity timestamp
func (s *Session) UpdateActivity() {
	now := time.Now()
	s.LastActivityAt = &now
	s.UpdatedAt = now
}

// SessionFilter represents filter options for session queries
type SessionFilter struct {
	UserID     *uuid.UUID
	DeviceType *string
	IsRevoked  *bool
	Limit      int
	Offset     int
	Sort       string
	Order      string
}

// Client represents an OAuth2 client application
type Client struct {
	ID               uuid.UUID `json:"id" db:"id"`
	TenantID         uuid.UUID `json:"tenant_id" db:"tenant_id"`
	ClientID         string    `json:"client_id" db:"client_id"`
	ClientSecretHash string    `json:"-" db:"client_secret_hash"`
	Name             string    `json:"name" db:"name"`
	RedirectURIs     []string  `json:"redirect_uris" db:"redirect_uris"`
	GrantTypes       []string  `json:"grant_types" db:"grant_types"`
	Scope            []string  `json:"scope,omitempty" db:"scope"`
	IsActive         bool      `json:"is_active" db:"is_active"`
	AccessTokenTTL   int       `json:"access_token_ttl" db:"access_token_ttl"`   // in seconds
	RefreshTokenTTL  int       `json:"refresh_token_ttl" db:"refresh_token_ttl"` // in seconds
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// IdentityProvider represents external OAuth identity provider configuration
type IdentityProvider struct {
	ID                    uuid.UUID `json:"id" db:"id"`
	TenantID              uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Provider              string    `json:"provider" db:"provider"` // google, github, facebook, etc.
	ClientID              string    `json:"client_id" db:"client_id"`
	ClientSecret          string    `json:"-" db:"client_secret"` // encrypted
	AuthorizationEndpoint *string   `json:"authorization_endpoint,omitempty" db:"authorization_endpoint"`
	TokenEndpoint         *string   `json:"token_endpoint,omitempty" db:"token_endpoint"`
	UserinfoEndpoint      *string   `json:"userinfo_endpoint,omitempty" db:"userinfo_endpoint"`
	Scopes                []string  `json:"scopes,omitempty" db:"scopes"`
	IsActive              bool      `json:"is_active" db:"is_active"`
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time `json:"updated_at" db:"updated_at"`
}
