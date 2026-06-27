package domain

import (
	"time"

	"github.com/google/uuid"
)

// UserStatus represents user account status
type UserStatus int

const (
	UserStatusBlocked UserStatus = 0
	UserStatusActive  UserStatus = 1
	UserStatusPending UserStatus = 2
)

func (s UserStatus) String() string {
	switch s {
	case UserStatusBlocked:
		return "BLOCKED"
	case UserStatusActive:
		return "ACTIVE"
	case UserStatusPending:
		return "PENDING"
	default:
		return "UNKNOWN"
	}
}

// User represents a user entity
type User struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	TenantID        uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	FullName        string     `json:"full_name" db:"full_name"`
	Email           string     `json:"email" db:"email"`
	Username        string     `json:"username" db:"username"`
	Phone           *string    `json:"phone,omitempty" db:"phone"`
	AvatarURL       *string    `json:"avatar_url,omitempty" db:"avatar_url"`
	IsEmailVerified bool       `json:"is_email_verified" db:"is_email_verified"`
	IsPhoneVerified bool       `json:"is_phone_verified" db:"is_phone_verified"`
	Status          UserStatus `json:"status" db:"status"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedBy       *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy       *uuid.UUID `json:"updated_by,omitempty" db:"updated_by"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	// Relations
	Roles       []Role       `json:"roles,omitempty" db:"-"`
	Credentials []Credential `json:"credentials,omitempty" db:"-"`
	Profile     *Profile     `json:"profile,omitempty" db:"-"`
	Attributes  []Attribute  `json:"attributes,omitempty" db:"-"`
}

// TableName returns the table name
func (User) TableName() string {
	return "users"
}

// IsActive checks if user is active
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

// Credential represents user authentication credentials
type Credential struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	UserID         uuid.UUID       `json:"user_id" db:"user_id"`
	Type           CredentialType  `json:"type" db:"type"`
	CredentialHash *string         `json:"credential_hash,omitempty" db:"credential_hash"`
	ProviderUserID *string         `json:"provider_user_id,omitempty" db:"provider_user_id"`
	ProviderData   *map[string]interface{} `json:"provider_data,omitempty" db:"provider_data"`
	IsActive       bool            `json:"is_active" db:"is_active"`
	LastUsedAt     *time.Time      `json:"last_used_at,omitempty" db:"last_used_at"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`
}

// CredentialType represents authentication credential type
type CredentialType string

const (
	CredentialTypePassword    CredentialType = "PASSWORD"
	CredentialTypeOAuthGoogle CredentialType = "OAUTH_GOOGLE"
	CredentialTypeOAuthGithub CredentialType = "OAUTH_GITHUB"
	CredentialTypeLDAP        CredentialType = "LDAP"
)

// Profile represents extended user profile
type Profile struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	Avatar      *string                `json:"avatar,omitempty" db:"avatar"`
	UserID      uuid.UUID              `json:"user_id" db:"user_id"`
	FullName    *string                `json:"full_name,omitempty" db:"full_name"`
	NIKNumber   *string                `json:"nik_number,omitempty" db:"nik_number"`
	PlaceBirth  *string                `json:"place_birth,omitempty" db:"place_birth"`
	DateOfBirth *time.Time             `json:"date_of_birth,omitempty" db:"date_of_birth"`
	Mobile      *string                `json:"mobile,omitempty" db:"mobile"`
	Address     *string                `json:"address,omitempty" db:"address"`
	ZipCode     *string                `json:"zip_code,omitempty" db:"zip_code"`
	Extras      map[string]interface{} `json:"extras,omitempty" db:"extras"`
}

// Attribute represents dynamic user attributes
type Attribute struct {
	ID        uuid.UUID              `json:"id" db:"id"`
	UserID    uuid.UUID              `json:"user_id" db:"user_id"`
	Key       string                 `json:"key" db:"key"`
	Value     map[string]interface{} `json:"value" db:"value"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt time.Time              `json:"updated_at" db:"updated_at"`
}

// UserFilter represents filter options for user queries
type UserFilter struct {
	TenantID *uuid.UUID
	Status   *UserStatus
	Search   *string
	RoleID   *uuid.UUID
	Limit    int
	Offset   int
	Sort     string
	Order    string
}
