package usecase

import (
	"context"
	"time"

	appErrors "github.com/anto1290/qlxion-monorepo/pkg/errors"
	"github.com/anto1290/qlxion-monorepo/services/auth-service/internal/domain"
	"github.com/anto1290/qlxion-monorepo/services/auth-service/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// UserUsecase handles user management business logic
type UserUsecase struct {
	userRepo   repository.UserRepository
	roleRepo   repository.RoleRepository
	tenantRepo repository.TenantRepository
	auditRepo  repository.AuditRepository
}

// NewUserUsecase creates a new UserUsecase
func NewUserUsecase(
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	tenantRepo repository.TenantRepository,
	auditRepo repository.AuditRepository,
) *UserUsecase {
	return &UserUsecase{
		userRepo:   userRepo,
		roleRepo:   roleRepo,
		tenantRepo: tenantRepo,
		auditRepo:  auditRepo,
	}
}

// CreateUser creates a new user (admin only)
func (u *UserUsecase) CreateUser(ctx context.Context, req CreateUserRequest) (*domain.User, error) {
	// Check tenant
	tenant, err := u.tenantRepo.GetByID(ctx, req.TenantID)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrNotFound, "Tenant not found", err)
	}

	if !tenant.IsActive() {
		return nil, appErrors.New(appErrors.ErrTenantInactive, "Tenant is not active")
	}

	// Check if email exists
	existing, err := u.userRepo.GetByEmailAndTenant(ctx, req.Email, req.TenantID)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to check email", err)
	}
	if existing != nil {
		return nil, appErrors.New(appErrors.ErrConflict, "Email already registered")
	}

	// Check if username exists
	existing, err = u.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to check username", err)
	}
	if existing != nil {
		return nil, appErrors.New(appErrors.ErrConflict, "Username already taken")
	}

	now := time.Now()
	user := &domain.User{
		ID:              uuid.New(),
		TenantID:        req.TenantID,
		FullName:        req.FullName,
		Email:           req.Email,
		Username:        req.Username,
		Phone:           req.Phone,
		Status:          domain.UserStatus(req.Status),
		IsEmailVerified: req.IsEmailVerified,
		CreatedBy:       req.CreatedBy,
		UpdatedBy:       req.CreatedBy,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if user.Status == 0 {
		user.Status = domain.UserStatusActive
	}

	if err := u.userRepo.Create(ctx, user); err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to create user", err)
	}

	// Create password credential if provided
	if req.Password != nil && *req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to hash password", err)
		}

		hashStr := string(hashedPassword)
		cred := &domain.Credential{
			ID:             uuid.New(),
			UserID:         user.ID,
			Type:           domain.CredentialTypePassword,
			CredentialHash: &hashStr,
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		if err := u.userRepo.CreateCredential(ctx, cred); err != nil {
			return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to create credentials", err)
		}
	}

	// Assign roles if provided
	for _, roleID := range req.RoleIDs {
		u.roleRepo.AssignRoleToUser(ctx, user.ID, roleID, req.CreatedBy, nil)
	}

	// Log audit
	u.logAudit(ctx, domain.AuditActionUserCreate, user, &req.TenantID, map[string]interface{}{
		"created_by": req.CreatedBy,
	})

	return user, nil
}

// GetUser gets a user by ID
func (u *UserUsecase) GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := u.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrNotFound, "User not found", err)
	}

	// Get roles
	roles, err := u.roleRepo.GetRolesByUserID(ctx, id)
	if err == nil {
		user.Roles = roles
	}

	// Get profile
	profile, err := u.userRepo.GetProfileByUserID(ctx, id)
	if err == nil && profile != nil {
		user.Profile = profile
	}

	return user, nil
}

// ListUsers lists users with filter
func (u *UserUsecase) ListUsers(ctx context.Context, filter domain.UserFilter) ([]domain.User, int64, error) {
	return u.userRepo.List(ctx, filter)
}

// UpdateUser updates a user
func (u *UserUsecase) UpdateUser(ctx context.Context, id uuid.UUID, req UpdateUserRequest) (*domain.User, error) {
	user, err := u.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrNotFound, "User not found", err)
	}

	// Update fields
	if req.FullName != nil {
		user.FullName = *req.FullName
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.Phone != nil {
		user.Phone = req.Phone
	}
	if req.AvatarURL != nil {
		user.AvatarURL = req.AvatarURL
	}
	if req.Status != nil {
		user.Status = domain.UserStatus(*req.Status)
	}
	if req.IsEmailVerified != nil {
		user.IsEmailVerified = *req.IsEmailVerified
	}

	user.UpdatedBy = req.UpdatedBy
	user.UpdatedAt = time.Now()

	if err := u.userRepo.Update(ctx, user); err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to update user", err)
	}

	// Update profile if provided
	if req.Profile != nil {
		req.Profile.UserID = id
		if err := u.userRepo.CreateOrUpdateProfile(ctx, req.Profile); err != nil {
			// Log but don't fail
		}
	}

	// Log audit
	u.logAudit(ctx, domain.AuditActionUserUpdate, user, &user.TenantID, map[string]interface{}{
		"updated_by": req.UpdatedBy,
	})

	return user, nil
}

// DeleteUser soft deletes a user
func (u *UserUsecase) DeleteUser(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	user, err := u.userRepo.GetByID(ctx, id)
	if err != nil {
		return appErrors.Wrap(appErrors.ErrNotFound, "User not found", err)
	}

	if err := u.userRepo.Delete(ctx, id, deletedBy); err != nil {
		return appErrors.Wrap(appErrors.ErrInternal, "Failed to delete user", err)
	}

	// Log audit
	u.logAudit(ctx, domain.AuditActionUserDelete, user, &user.TenantID, map[string]interface{}{
		"deleted_by": deletedBy,
	})

	return nil
}

// GetUserRoles gets roles for a user
func (u *UserUsecase) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]domain.Role, error) {
	return u.roleRepo.GetRolesByUserID(ctx, userID)
}

// AssignRole assigns a role to a user
func (u *UserUsecase) AssignRole(ctx context.Context, userID, roleID uuid.UUID, assignedBy *uuid.UUID) error {
	if err := u.roleRepo.AssignRoleToUser(ctx, userID, roleID, assignedBy, nil); err != nil {
		return appErrors.Wrap(appErrors.ErrInternal, "Failed to assign role", err)
	}

	// Log audit
	u.logAudit(ctx, domain.AuditActionRoleAssign, &domain.User{ID: userID}, nil, map[string]interface{}{
		"role_id":     roleID,
		"assigned_by": assignedBy,
	})

	return nil
}

// RemoveRole removes a role from a user
func (u *UserUsecase) RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error {
	if err := u.roleRepo.RemoveRoleFromUser(ctx, userID, roleID); err != nil {
		return appErrors.Wrap(appErrors.ErrInternal, "Failed to remove role", err)
	}

	// Log audit
	u.logAudit(ctx, domain.AuditActionRoleRevoke, &domain.User{ID: userID}, nil, map[string]interface{}{
		"role_id": roleID,
	})

	return nil
}

// Requests

type CreateUserRequest struct {
	TenantID        uuid.UUID
	FullName        string
	Email           string
	Username        string
	Password        *string
	Phone           *string
	Status          int
	IsEmailVerified bool
	RoleIDs         []uuid.UUID
	CreatedBy       *uuid.UUID
}

type UpdateUserRequest struct {
	FullName        *string
	Email           *string
	Phone           *string
	AvatarURL       *string
	Status          *int
	IsEmailVerified *bool
	Profile         *domain.Profile
	UpdatedBy       *uuid.UUID
}

func (u *UserUsecase) logAudit(ctx context.Context, action domain.AuditAction, user *domain.User, tenantID *uuid.UUID, details map[string]interface{}) {
	log := &domain.AuditLog{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    &user.ID,
		Action:    action,
		Details:   details,
		CreatedAt: time.Now(),
	}

	if err := u.auditRepo.Create(ctx, log); err != nil {
		// Log error but don't fail
	}
}
