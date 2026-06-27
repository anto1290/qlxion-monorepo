package usecase

import (
	"context"
	"time"

	appErrors "github.com/anto1290/qlxion-monorepo/pkg/errors"
	"github.com/anto1290/qlxion-monorepo/services/auth-service/internal/domain"
	"github.com/anto1290/qlxion-monorepo/services/auth-service/internal/repository"
	"github.com/google/uuid"
)

// TenantUsecase handles tenant management business logic
type TenantUsecase struct {
	tenantRepo repository.TenantRepository
	auditRepo  repository.AuditRepository
}

// NewTenantUsecase creates a new TenantUsecase
func NewTenantUsecase(
	tenantRepo repository.TenantRepository,
	auditRepo repository.AuditRepository,
) *TenantUsecase {
	return &TenantUsecase{
		tenantRepo: tenantRepo,
		auditRepo:  auditRepo,
	}
}

// CreateTenant creates a new tenant
func (u *TenantUsecase) CreateTenant(ctx context.Context, req CreateTenantRequest) (*domain.Tenant, error) {
	// Check if code already exists
	existing, err := u.tenantRepo.GetByCode(ctx, req.Code)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to check tenant code", err)
	}
	if existing != nil {
		return nil, appErrors.New(appErrors.ErrConflict, "Tenant code already exists")
	}

	// Check if domain already exists
	if req.Domain != nil && *req.Domain != "" {
		existing, err = u.tenantRepo.GetByDomain(ctx, *req.Domain)
		if err != nil {
			return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to check tenant domain", err)
		}
		if existing != nil {
			return nil, appErrors.New(appErrors.ErrConflict, "Tenant domain already exists")
		}
	}

	now := time.Now()
	tenant := &domain.Tenant{
		ID:        uuid.New(),
		Code:      req.Code,
		Name:      req.Name,
		Domain:    req.Domain,
		Config:    req.Config,
		Status:    domain.TenantStatusActive,
		CreatedBy: req.CreatedBy,
		UpdatedBy: req.CreatedBy,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := u.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to create tenant", err)
	}

	// Log audit
	u.logAudit(ctx, domain.AuditActionTenantCreate, req.CreatedBy, &tenant.ID, map[string]interface{}{
		"code": req.Code,
		"name": req.Name,
	})

	return tenant, nil
}

// GetTenant gets a tenant by ID
func (u *TenantUsecase) GetTenant(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	tenant, err := u.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrNotFound, "Tenant not found", err)
	}
	return tenant, nil
}

// GetTenantByCode gets a tenant by code
func (u *TenantUsecase) GetTenantByCode(ctx context.Context, code string) (*domain.Tenant, error) {
	tenant, err := u.tenantRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrNotFound, "Tenant not found", err)
	}
	if tenant == nil {
		return nil, appErrors.New(appErrors.ErrNotFound, "Tenant not found")
	}
	return tenant, nil
}

// ListTenants lists tenants with filter
func (u *TenantUsecase) ListTenants(ctx context.Context, filter domain.TenantFilter) ([]domain.Tenant, int64, error) {
	return u.tenantRepo.List(ctx, filter)
}

// UpdateTenant updates a tenant
func (u *TenantUsecase) UpdateTenant(ctx context.Context, id uuid.UUID, req UpdateTenantRequest) (*domain.Tenant, error) {
	tenant, err := u.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrNotFound, "Tenant not found", err)
	}

	if req.Name != nil {
		tenant.Name = *req.Name
	}
	if req.Domain != nil {
		tenant.Domain = req.Domain
	}
	if req.Config != nil {
		tenant.Config = req.Config
	}
	if req.Status != nil {
		tenant.Status = domain.TenantStatus(*req.Status)
	}

	tenant.UpdatedBy = req.UpdatedBy
	tenant.UpdatedAt = time.Now()

	if err := u.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, appErrors.Wrap(appErrors.ErrInternal, "Failed to update tenant", err)
	}

	// Log audit
	u.logAudit(ctx, domain.AuditActionTenantUpdate, req.UpdatedBy, &id, map[string]interface{}{
		"code": tenant.Code,
	})

	return tenant, nil
}

// DeleteTenant soft deletes a tenant
func (u *TenantUsecase) DeleteTenant(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	tenant, err := u.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return appErrors.Wrap(appErrors.ErrNotFound, "Tenant not found", err)
	}

	if err := u.tenantRepo.Delete(ctx, id, deletedBy); err != nil {
		return appErrors.Wrap(appErrors.ErrInternal, "Failed to delete tenant", err)
	}

	// Log audit
	u.logAudit(ctx, domain.AuditActionTenantUpdate, &deletedBy, &tenant.ID, map[string]interface{}{
		"code":   tenant.Code,
		"action": "delete",
	})

	return nil
}

// Requests

type CreateTenantRequest struct {
	Code      string
	Name      string
	Domain    *string
	Config    map[string]interface{}
	CreatedBy *uuid.UUID
}

type UpdateTenantRequest struct {
	Name      *string
	Domain    *string
	Config    map[string]interface{}
	Status    *int
	UpdatedBy *uuid.UUID
}

func (u *TenantUsecase) logAudit(ctx context.Context, action domain.AuditAction, userID *uuid.UUID, tenantID *uuid.UUID, details map[string]interface{}) {
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
