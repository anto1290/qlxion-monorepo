package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	appErrors "github.com/qlxion/qlxion-monorepo/pkg/errors"
	"github.com/qlxion/qlxion-monorepo/services/auth-service/internal/domain"
	"github.com/qlxion/qlxion-monorepo/services/auth-service/internal/repository"
)

// RoleUsecase handles role and permission management business logic
type RoleUsecase struct {
	roleRepo   repository.RoleRepository
	userRepo   repository.UserRepository
	auditRepo  repository.AuditRepository
}

// NewRoleUsecase creates a new RoleUsecase
func NewRoleUsecase(
	roleRepo repository.RoleRepository,
	userRepo repository.UserRepository,
	auditRepo repository.AuditRepository,
) *RoleUsecase {
	return &RoleUsecase{
		roleRepo:   roleRepo,
		userRepo:   userRepo,
		auditRepo:  auditRepo,
	}
}

// CreateRole creates a new role
func (u *RoleUsecase) CreateRole(ctx context.Context, req CreateRoleRequest) (*domain.Role, error) {
	// Check if role code already exists
	existing, err := u.roleRepo.GetByCode(ctx, req.Code, req.TenantID)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to check role code", err)
	}
	if existing != nil {
		return nil, appErrors.New(appErrors.ErrConflict, "Role code already exists")
	}

	now := time.Now()
	role := &domain.Role{
		ID:              uuid.New(),
		TenantID:        req.TenantID,
		Code:            req.Code,
		Name:            req.Name,
		Description:     req.Description,
		IsSystemDefined: false,
		Status:          domain.RoleStatusActive,
		CreatedBy:       req.CreatedBy,
		UpdatedBy:       req.CreatedBy,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := u.roleRepo.Create(ctx, role); err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to create role", err)
	}

	// Assign permissions if provided
	for _, permID := range req.PermissionIDs {
		u.roleRepo.AssignPermission(ctx, role.ID, permID, req.CreatedBy)
	}

	// Log audit
	u.logAudit(ctx, domain.AuditActionPermissionChange, req.CreatedBy, req.TenantID, map[string]interface{}{
		"action":   "create_role",
		"role_code": req.Code,
	})

	return role, nil
}

// GetRole gets a role by ID
func (u *RoleUsecase) GetRole(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	role, err := u.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrNotFound, "Role not found", err)
	}

	// Get permissions
	permissions, err := u.roleRepo.GetPermissionsByRoleID(ctx, id)
	if err == nil {
		role.Permissions = permissions
	}

	return role, nil
}

// ListRoles lists roles with filter
func (u *RoleUsecase) ListRoles(ctx context.Context, filter domain.RoleFilter) ([]domain.Role, int64, error) {
	return u.roleRepo.List(ctx, filter)
}

// UpdateRole updates a role
func (u *RoleUsecase) UpdateRole(ctx context.Context, id uuid.UUID, req UpdateRoleRequest) (*domain.Role, error) {
	role, err := u.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrNotFound, "Role not found", err)
	}

	if role.IsSystemDefined {
		return nil, appErrors.New(appErrors.ErrForbidden, "System roles cannot be modified")
	}

	if req.Name != nil {
		role.Name = *req.Name
	}
	if req.Description != nil {
		role.Description = req.Description
	}
	if req.Status != nil {
		role.Status = domain.RoleStatus(*req.Status)
	}

	role.UpdatedBy = req.UpdatedBy
	role.UpdatedAt = time.Now()

	if err := u.roleRepo.Update(ctx, role); err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to update role", err)
	}

	// Log audit
	u.logAudit(ctx, domain.AuditActionPermissionChange, req.UpdatedBy, role.TenantID, map[string]interface{}{
		"action":    "update_role",
		"role_code": role.Code,
	})

	return role, nil
}

// DeleteRole deletes a role
func (u *RoleUsecase) DeleteRole(ctx context.Context, id uuid.UUID) error {
	role, err := u.roleRepo.GetByID(ctx, id)
	if err != nil {
		return appErrors.Wrap(appErrors.ErrNotFound, "Role not found", err)
	}

	if role.IsSystemDefined {
		return appErrors.New(appErrors.ErrForbidden, "System roles cannot be deleted")
	}

	if err := u.roleRepo.Delete(ctx, id); err != nil {
		return appErrors.Wrap(appErrors.ErrInternal, "Failed to delete role", err)
	}

	// Log audit
	u.logAudit(ctx, domain.AuditActionPermissionChange, nil, role.TenantID, map[string]interface{}{
		"action":    "delete_role",
		"role_code": role.Code,
	})

	return nil
}

// GetPermissions returns all permissions
func (u *RoleUsecase) GetPermissions(ctx context.Context) ([]domain.Permission, error) {
	return u.roleRepo.GetAllPermissions(ctx)
}

// CreatePermission creates a new permission
func (u *RoleUsecase) CreatePermission(ctx context.Context, req CreatePermissionRequest) (*domain.Permission, error) {
	now := time.Now()
	perm := &domain.Permission{
		ID:              uuid.New(),
		Resource:        req.Resource,
		Action:          req.Action,
		Description:     req.Description,
		IsSystemDefined: false,
		CreatedAt:       now,
	}

	if err := u.roleRepo.CreatePermission(ctx, perm); err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to create permission", err)
	}

	return perm, nil
}

// AssignPermission assigns a permission to a role
func (u *RoleUsecase) AssignPermission(ctx context.Context, roleID, permissionID uuid.UUID, assignedBy *uuid.UUID) error {
	if err := u.roleRepo.AssignPermission(ctx, roleID, permissionID, assignedBy); err != nil {
		return appErrors.Wrap(appErrors.ErrInternal, "Failed to assign permission", err)
	}

	// Log audit
	u.logAudit(ctx, domain.AuditActionPermissionChange, assignedBy, nil, map[string]interface{}{
		"action":       "assign_permission",
		"role_id":      roleID,
		"permission_id": permissionID,
	})

	return nil
}

// RemovePermission removes a permission from a role
func (u *RoleUsecase) RemovePermission(ctx context.Context, roleID, permissionID uuid.UUID) error {
	if err := u.roleRepo.RemovePermission(ctx, roleID, permissionID); err != nil {
		return appErrors.Wrap(appErrors.ErrInternal, "Failed to remove permission", err)
	}

	// Log audit
	u.logAudit(ctx, domain.AuditActionPermissionChange, nil, nil, map[string]interface{}{
		"action":        "remove_permission",
		"role_id":       roleID,
		"permission_id": permissionID,
	})

	return nil
}

// GetRolePermissions gets permissions for a role
func (u *RoleUsecase) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]domain.Permission, error) {
	return u.roleRepo.GetPermissionsByRoleID(ctx, roleID)
}

// Requests

type CreateRoleRequest struct {
	Code          string
	Name          string
	Description   *string
	TenantID      *uuid.UUID
	PermissionIDs []uuid.UUID
	CreatedBy     *uuid.UUID
}

type UpdateRoleRequest struct {
	Name        *string
	Description *string
	Status      *int
	UpdatedBy   *uuid.UUID
}

type CreatePermissionRequest struct {
	Resource    string
	Action      string
	Description *string
}

func (u *RoleUsecase) logAudit(ctx context.Context, action domain.AuditAction, userID *uuid.UUID, tenantID *uuid.UUID, details map[string]interface{}) {
	log := &domain.AuditLog{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    userID,
		Action:    action,
		Details:   details,
		CreatedAt: time.Now(),
	}

	if err := u.auditRepo.Create(ctx, log); err != nil {
		// Log error but don't fail
	}
}
