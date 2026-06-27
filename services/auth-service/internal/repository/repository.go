package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/qlxion/qlxion-monorepo/services/auth-service/internal/domain"
)

// UserRepository defines user repository operations
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	GetByEmailAndTenant(ctx context.Context, email string, tenantID uuid.UUID) (*domain.User, error)
	List(ctx context.Context, filter domain.UserFilter) ([]domain.User, int64, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
	
	// Credential operations
	CreateCredential(ctx context.Context, cred *domain.Credential) error
	GetCredentialsByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Credential, error)
	GetActiveCredentialByType(ctx context.Context, userID uuid.UUID, credType domain.CredentialType) (*domain.Credential, error)
	UpdateCredential(ctx context.Context, cred *domain.Credential) error
	
	// Profile operations
	GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*domain.Profile, error)
	CreateOrUpdateProfile(ctx context.Context, profile *domain.Profile) error
	
	// Attribute operations
	GetAttributesByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Attribute, error)
	SetAttribute(ctx context.Context, attr *domain.Attribute) error
}

// RoleRepository defines role repository operations
type RoleRepository interface {
	Create(ctx context.Context, role *domain.Role) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Role, error)
	GetByCode(ctx context.Context, code string, tenantID *uuid.UUID) (*domain.Role, error)
	List(ctx context.Context, filter domain.RoleFilter) ([]domain.Role, int64, error)
	Update(ctx context.Context, role *domain.Role) error
	Delete(ctx context.Context, id uuid.UUID) error
	
	// Permission operations
	GetPermissionsByRoleID(ctx context.Context, roleID uuid.UUID) ([]domain.Permission, error)
	AssignPermission(ctx context.Context, roleID, permissionID uuid.UUID, assignedBy *uuid.UUID) error
	RemovePermission(ctx context.Context, roleID, permissionID uuid.UUID) error
	GetAllPermissions(ctx context.Context) ([]domain.Permission, error)
	CreatePermission(ctx context.Context, perm *domain.Permission) error
	
	// User role operations
	GetRolesByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Role, error)
	AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID, assignedBy *uuid.UUID, expiresAt *interface{}) error
	RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error
	GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]domain.Permission, error)
}

// TenantRepository defines tenant repository operations
type TenantRepository interface {
	Create(ctx context.Context, tenant *domain.Tenant) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error)
	GetByCode(ctx context.Context, code string) (*domain.Tenant, error)
	GetByDomain(ctx context.Context, domain string) (*domain.Tenant, error)
	List(ctx context.Context, filter domain.TenantFilter) ([]domain.Tenant, int64, error)
	Update(ctx context.Context, tenant *domain.Tenant) error
	Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
}

// SessionRepository defines session repository operations
type SessionRepository interface {
	Create(ctx context.Context, session *domain.Session) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Session, error)
	GetByRefreshTokenHash(ctx context.Context, hash string) (*domain.Session, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, filter domain.SessionFilter) ([]domain.Session, int64, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeAllUserSessions(ctx context.Context, userID uuid.UUID, exceptSessionID *uuid.UUID) error
	UpdateActivity(ctx context.Context, id uuid.UUID) error
	
	// Client operations
	CreateClient(ctx context.Context, client *domain.Client) error
	GetClientByClientID(ctx context.Context, clientID string) (*domain.Client, error)
	
	// Identity Provider operations
	CreateIdentityProvider(ctx context.Context, provider *domain.IdentityProvider) error
	GetIdentityProvidersByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.IdentityProvider, error)
	GetIdentityProviderByTenantAndProvider(ctx context.Context, tenantID uuid.UUID, provider string) (*domain.IdentityProvider, error)
}

// AuditRepository defines audit log repository operations
type AuditRepository interface {
	Create(ctx context.Context, log *domain.AuditLog) error
	List(ctx context.Context, filter domain.AuditLogFilter) ([]domain.AuditLog, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.AuditLog, error)
}
