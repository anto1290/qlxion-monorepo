package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/qlxion/qlxion-monorepo/services/auth-service/internal/domain"
)

// TenantRepo implements TenantRepository
type TenantRepo struct {
	db *pgxpool.Pool
}

// NewTenantRepo creates a new TenantRepo
func NewTenantRepo(db *pgxpool.Pool) *TenantRepo {
	return &TenantRepo{db: db}
}

// Create creates a new tenant
func (r *TenantRepo) Create(ctx context.Context, tenant *domain.Tenant) error {
	query := `
		INSERT INTO tenants (id, code, name, domain, config, status, created_by, updated_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	
	_, err := r.db.Exec(ctx, query,
		tenant.ID, tenant.Code, tenant.Name, tenant.Domain,
		tenant.Config, tenant.Status, tenant.CreatedBy, tenant.UpdatedBy,
		tenant.CreatedAt, tenant.UpdatedAt,
	)
	return err
}

// GetByID gets tenant by ID
func (r *TenantRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	query := `
		SELECT id, code, name, domain, config, status,
			created_by, updated_by, created_at, updated_at, deleted_at
		FROM tenants WHERE id = $1 AND deleted_at IS NULL
	`
	
	tenant := &domain.Tenant{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&tenant.ID, &tenant.Code, &tenant.Name, &tenant.Domain,
		&tenant.Config, &tenant.Status, &tenant.CreatedBy, &tenant.UpdatedBy,
		&tenant.CreatedAt, &tenant.UpdatedAt, &tenant.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("tenant not found")
		}
		return nil, err
	}
	return tenant, nil
}

// GetByCode gets tenant by code
func (r *TenantRepo) GetByCode(ctx context.Context, code string) (*domain.Tenant, error) {
	query := `
		SELECT id, code, name, domain, config, status,
			created_by, updated_by, created_at, updated_at, deleted_at
		FROM tenants WHERE code = $1 AND deleted_at IS NULL
	`
	
	tenant := &domain.Tenant{}
	err := r.db.QueryRow(ctx, query, code).Scan(
		&tenant.ID, &tenant.Code, &tenant.Name, &tenant.Domain,
		&tenant.Config, &tenant.Status, &tenant.CreatedBy, &tenant.UpdatedBy,
		&tenant.CreatedAt, &tenant.UpdatedAt, &tenant.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return tenant, nil
}

// GetByDomain gets tenant by domain
func (r *TenantRepo) GetByDomain(ctx context.Context, domain string) (*domain.Tenant, error) {
	query := `
		SELECT id, code, name, domain, config, status,
			created_by, updated_by, created_at, updated_at, deleted_at
		FROM tenants WHERE domain = $1 AND deleted_at IS NULL
	`
	
	tenant := &domain.Tenant{}
	err := r.db.QueryRow(ctx, query, domain).Scan(
		&tenant.ID, &tenant.Code, &tenant.Name, &tenant.Domain,
		&tenant.Config, &tenant.Status, &tenant.CreatedBy, &tenant.UpdatedBy,
		&tenant.CreatedAt, &tenant.UpdatedAt, &tenant.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return tenant, nil
}

// List lists tenants with filter
func (r *TenantRepo) List(ctx context.Context, filter domain.TenantFilter) ([]domain.Tenant, int64, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "deleted_at IS NULL")

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	if filter.Search != nil && *filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(code ILIKE $%d OR name ILIKE $%d OR domain ILIKE $%d)",
			argIdx, argIdx, argIdx,
		))
		args = append(args, "%"+*filter.Search+"%")
		argIdx++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int64
	countQuery := `SELECT COUNT(*) FROM tenants ` + whereClause
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	sort := filter.Sort
	if sort == "" {
		sort = "created_at"
	}
	order := filter.Order
	if order == "" {
		order = "DESC"
	}

	limit := filter.Limit
	if limit == 0 {
		limit = 20
	}

	query := fmt.Sprintf(`
		SELECT id, code, name, domain, config, status,
			created_by, updated_by, created_at, updated_at, deleted_at
		FROM tenants %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sort, order, argIdx, argIdx+1)

	args = append(args, limit, filter.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tenants []domain.Tenant
	for rows.Next() {
		var tenant domain.Tenant
		err := rows.Scan(
			&tenant.ID, &tenant.Code, &tenant.Name, &tenant.Domain,
			&tenant.Config, &tenant.Status, &tenant.CreatedBy, &tenant.UpdatedBy,
			&tenant.CreatedAt, &tenant.UpdatedAt, &tenant.DeletedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		tenants = append(tenants, tenant)
	}

	return tenants, total, nil
}

// Update updates a tenant
func (r *TenantRepo) Update(ctx context.Context, tenant *domain.Tenant) error {
	query := `
		UPDATE tenants SET
			code = $1, name = $2, domain = $3, config = $4,
			status = $5, updated_by = $6, updated_at = $7
		WHERE id = $8 AND deleted_at IS NULL
	`
	
	_, err := r.db.Exec(ctx, query,
		tenant.Code, tenant.Name, tenant.Domain, tenant.Config,
		tenant.Status, tenant.UpdatedBy, time.Now(), tenant.ID,
	)
	return err
}

// Delete soft deletes a tenant
func (r *TenantRepo) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	query := `
		UPDATE tenants SET
			status = $1, updated_by = $2, updated_at = $3, deleted_at = $4
		WHERE id = $5 AND deleted_at IS NULL
	`
	
	now := time.Now()
	_, err := r.db.Exec(ctx, query,
		domain.TenantStatusInactive, deletedBy, now, now, id,
	)
	return err
}
