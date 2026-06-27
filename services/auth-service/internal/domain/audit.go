package domain

import (
	"time"

	"github.com/google/uuid"
)

// AuditAction represents types of audit actions
type AuditAction string

const (
	AuditActionLogin              AuditAction = "LOGIN"
	AuditActionLogout             AuditAction = "LOGOUT"
	AuditActionRegister           AuditAction = "REGISTER"
	AuditActionPasswordChange     AuditAction = "PASSWORD_CHANGE"
	AuditActionPasswordReset      AuditAction = "PASSWORD_RESET"
	AuditActionRoleAssign         AuditAction = "ROLE_ASSIGN"
	AuditActionRoleRevoke         AuditAction = "ROLE_REVOKE"
	AuditActionPermissionChange   AuditAction = "PERMISSION_CHANGE"
	AuditActionUserCreate         AuditAction = "USER_CREATE"
	AuditActionUserUpdate         AuditAction = "USER_UPDATE"
	AuditActionUserDelete         AuditAction = "USER_DELETE"
	AuditActionTenantCreate       AuditAction = "TENANT_CREATE"
	AuditActionTenantUpdate       AuditAction = "TENANT_UPDATE"
	AuditActionSessionRevoke      AuditAction = "SESSION_REVOKE"
	AuditActionMFAEnabled         AuditAction = "MFA_ENABLED"
	AuditActionMFADisabled        AuditAction = "MFA_DISABLED"
	AuditActionTokenRefresh       AuditAction = "TOKEN_REFRESH"
	AuditActionAccountLocked      AuditAction = "ACCOUNT_LOCKED"
	AuditActionAccountUnlocked    AuditAction = "ACCOUNT_UNLOCKED"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID         uuid.UUID              `json:"id" db:"id"`
	TenantID   *uuid.UUID             `json:"tenant_id,omitempty" db:"tenant_id"`
	UserID     *uuid.UUID             `json:"user_id,omitempty" db:"user_id"`
	Action     AuditAction            `json:"action" db:"action"`
	Resource   *string                `json:"resource,omitempty" db:"resource"`
	ResourceID *uuid.UUID             `json:"resource_id,omitempty" db:"resource_id"`
	Details    map[string]interface{} `json:"details,omitempty" db:"details"`
	IPAddress  *string                `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent  *string                `json:"user_agent,omitempty" db:"user_agent"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
}

// TableName returns the table name
func (AuditLog) TableName() string {
	return "audit_logs"
}

// AuditLogFilter represents filter options for audit log queries
type AuditLogFilter struct {
	TenantID   *uuid.UUID
	UserID     *uuid.UUID
	Action     *AuditAction
	Resource   *string
	StartDate  *time.Time
	EndDate    *time.Time
	Limit      int
	Offset     int
	Sort       string
	Order      string
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email      string     `json:"email" validate:"required,email"`
	Password   string     `json:"password" validate:"required"`
	TenantID   *uuid.UUID `json:"tenant_id,omitempty"`
	DeviceName *string    `json:"device_name,omitempty"`
	DeviceType *string    `json:"device_type,omitempty"` // mobile, desktop, web
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	TenantID uuid.UUID `json:"tenant_id" validate:"required"`
	FullName string    `json:"full_name" validate:"required,max=255"`
	Email    string    `json:"email" validate:"required,email"`
	Username string    `json:"username" validate:"required,min=3,max=50"`
	Password string    `json:"password" validate:"required,min=8"`
	Phone    *string   `json:"phone,omitempty"`
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"` // seconds
	Scope        []string  `json:"scope,omitempty"`
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	User  *User      `json:"user"`
	Token *TokenPair `json:"token"`
}
