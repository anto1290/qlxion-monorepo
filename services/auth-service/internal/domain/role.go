package domain

import (
	"time"

	"github.com/google/uuid"
)

// Role represents a role entity
type Role struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	TenantID       *uuid.UUID `json:"tenant_id,omitempty" db:"tenant_id"`
	Code           string     `json:"code" db:"code"`
	Name           string     `json:"name" db:"name"`
	Description    *string    `json:"description,omitempty" db:"description"`
	IsSystemDefined bool      `json:"is_system_defined" db:"is_system_defined"`
	Status         RoleStatus `json:"status" db:"status"`
	CreatedBy      *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy      *uuid.UUID `json:"updated_by,omitempty" db:"updated_by"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`

	// Relations
	Permissions []Permission `json:"permissions,omitempty" db:"-"`
}

// RoleStatus represents role status
type RoleStatus int

const (
	RoleStatusInactive RoleStatus = 0
	RoleStatusActive   RoleStatus = 1
)

func (s RoleStatus) String() string {
	switch s {
	case RoleStatusInactive:
		return "INACTIVE"
	case RoleStatusActive:
		return "ACTIVE"
	default:
		return "UNKNOWN"
	}
}

// Permission represents a permission entity
type Permission struct {
	ID             uuid.UUID `json:"id" db:"id"`
	Resource       string    `json:"resource" db:"resource"`
	Action         string    `json:"action" db:"action"`
	Description    *string   `json:"description,omitempty" db:"description"`
	IsSystemDefined bool     `json:"is_system_defined" db:"is_system_defined"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// RolePermission represents the many-to-many relationship between roles and permissions
type RolePermission struct {
	RoleID       uuid.UUID  `json:"role_id" db:"role_id"`
	PermissionID uuid.UUID  `json:"permission_id" db:"permission_id"`
	AssignedAt   time.Time  `json:"assigned_at" db:"assigned_at"`
	AssignedBy   *uuid.UUID `json:"assigned_by,omitempty" db:"assigned_by"`
}

// UserRole represents the many-to-many relationship between users and roles
type UserRole struct {
	UserID     uuid.UUID  `json:"user_id" db:"user_id"`
	RoleID     uuid.UUID  `json:"role_id" db:"role_id"`
	AssignedAt time.Time  `json:"assigned_at" db:"assigned_at"`
	AssignedBy *uuid.UUID `json:"assigned_by,omitempty" db:"assigned_by"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty" db:"expires_at"`
}

// Predefined system roles
const (
	RoleSuperAdmin     = "ROLE_SUPER_ADMIN"
	RoleAdmin          = "ROLE_ADMIN"
	RoleUser           = "ROLE_USER"
	RoleTenantAdmin    = "ROLE_TENANT_ADMIN"
	RoleTenantManager  = "ROLE_TENANT_MANAGER"
)

// Predefined system permissions
const (
	PermissionUserCreate  = "user:create"
	PermissionUserRead    = "user:read"
	PermissionUserUpdate  = "user:update"
	PermissionUserDelete  = "user:delete"
	PermissionRoleCreate  = "role:create"
	PermissionRoleRead    = "role:read"
	PermissionRoleUpdate  = "role:update"
	PermissionRoleDelete  = "role:delete"
	PermissionTenantCreate = "tenant:create"
	PermissionTenantRead   = "tenant:read"
	PermissionTenantUpdate = "tenant:update"
	PermissionTenantDelete = "tenant:delete"
)

// RoleFilter represents filter options for role queries
type RoleFilter struct {
	TenantID *uuid.UUID
	Status   *RoleStatus
	Search   *string
	Limit    int
	Offset   int
	Sort     string
	Order    string
}
