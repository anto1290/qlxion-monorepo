package domain

import (
	"time"

	"github.com/google/uuid"
)

// TenantStatus represents tenant status
type TenantStatus int

const (
	TenantStatusInactive   TenantStatus = 0
	TenantStatusActive     TenantStatus = 1
	TenantStatusSuspended  TenantStatus = 2
)

func (s TenantStatus) String() string {
	switch s {
	case TenantStatusInactive:
		return "INACTIVE"
	case TenantStatusActive:
		return "ACTIVE"
	case TenantStatusSuspended:
		return "SUSPENDED"
	default:
		return "UNKNOWN"
	}
}

// Tenant represents a tenant entity (multi-tenant support)
type Tenant struct {
	ID        uuid.UUID              `json:"id" db:"id"`
	Code      string                 `json:"code" db:"code"`
	Name      string                 `json:"name" db:"name"`
	Domain    *string                `json:"domain,omitempty" db:"domain"`
	Config    map[string]interface{} `json:"config,omitempty" db:"config"`
	Status    TenantStatus           `json:"status" db:"status"`
	CreatedBy *uuid.UUID             `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy *uuid.UUID             `json:"updated_by,omitempty" db:"updated_by"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt time.Time              `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time             `json:"deleted_at,omitempty" db:"deleted_at"`
}

// TableName returns the table name
func (Tenant) TableName() string {
	return "tenants"
}

// IsActive checks if tenant is active
func (t *Tenant) IsActive() bool {
	return t.Status == TenantStatusActive
}

// TenantConfig represents tenant-specific configuration
type TenantConfig struct {
	SessionTimeout      int  `json:"session_timeout,omitempty"`       // in minutes
	MaxLoginAttempts    int  `json:"max_login_attempts,omitempty"`
	PasswordMinLength   int  `json:"password_min_length,omitempty"`
	RequireStrongPassword bool `json:"require_strong_password,omitempty"`
	AllowRegistration   bool `json:"allow_registration,omitempty"`
	MFARequired         bool `json:"mfa_required,omitempty"`
}

// TenantFilter represents filter options for tenant queries
type TenantFilter struct {
	Status *TenantStatus
	Search *string
	Limit  int
	Offset int
	Sort   string
	Order  string
}
