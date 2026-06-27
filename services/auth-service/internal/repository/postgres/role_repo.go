package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/anto1290/qlxion-monorepo/services/auth-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RoleRepo implements RoleRepository
type RoleRepo struct {
	db *pgxpool.Pool
}

// NewRoleRepo creates a new RoleRepo
func NewRoleRepo(db *pgxpool.Pool) *RoleRepo {
	return &RoleRepo{db: db}
}

// Create creates a new role
func (r *RoleRepo) Create(ctx context.Context, role *domain.Role) error {
	query := `
		INSERT INTO roles (id, tenant_id, code, name, description, is_system_defined, status, created_by, updated_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.Exec(ctx, query,
		role.ID, role.TenantID, role.Code, role.Name, role.Description,
		role.IsSystemDefined, role.Status, role.CreatedBy, role.UpdatedBy,
		role.CreatedAt, role.UpdatedAt,
	)
	return err
}

// GetByID gets role by ID
func (r *RoleRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	query := `
		SELECT id, tenant_id, code, name, description, is_system_defined, status,
			created_by, updated_by, created_at, updated_at
		FROM roles WHERE id = $1
	`

	role := &domain.Role{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&role.ID, &role.TenantID, &role.Code, &role.Name, &role.Description,
		&role.IsSystemDefined, &role.Status, &role.CreatedBy, &role.UpdatedBy,
		&role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("role not found")
		}
		return nil, err
	}
	return role, nil
}

// GetByCode gets role by code
func (r *RoleRepo) GetByCode(ctx context.Context, code string, tenantID *uuid.UUID) (*domain.Role, error) {
	var query string
	var args []interface{}

	if tenantID != nil {
		query = `SELECT id, tenant_id, code, name, description, is_system_defined, status,
			created_by, updated_by, created_at, updated_at
			FROM roles WHERE code = $1 AND (tenant_id = $2 OR tenant_id IS NULL)`
		args = append(args, code, *tenantID)
	} else {
		query = `SELECT id, tenant_id, code, name, description, is_system_defined, status,
			created_by, updated_by, created_at, updated_at
			FROM roles WHERE code = $1 AND tenant_id IS NULL`
		args = append(args, code)
	}

	role := &domain.Role{}
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&role.ID, &role.TenantID, &role.Code, &role.Name, &role.Description,
		&role.IsSystemDefined, &role.Status, &role.CreatedBy, &role.UpdatedBy,
		&role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return role, nil
}

// List lists roles with filter
func (r *RoleRepo) List(ctx context.Context, filter domain.RoleFilter) ([]domain.Role, int64, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.TenantID != nil {
		conditions = append(conditions, fmt.Sprintf("(tenant_id = $%d OR tenant_id IS NULL)", argIdx))
		args = append(args, *filter.TenantID)
		argIdx++
	}

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	if filter.Search != nil && *filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(code ILIKE $%d OR name ILIKE $%d)", argIdx, argIdx,
		))
		args = append(args, "%"+*filter.Search+"%")
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int64
	countQuery := `SELECT COUNT(*) FROM roles ` + whereClause
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
		SELECT id, tenant_id, code, name, description, is_system_defined, status,
			created_by, updated_by, created_at, updated_at
		FROM roles %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sort, order, argIdx, argIdx+1)

	args = append(args, limit, filter.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var roles []domain.Role
	for rows.Next() {
		var role domain.Role
		err := rows.Scan(
			&role.ID, &role.TenantID, &role.Code, &role.Name, &role.Description,
			&role.IsSystemDefined, &role.Status, &role.CreatedBy, &role.UpdatedBy,
			&role.CreatedAt, &role.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		roles = append(roles, role)
	}

	return roles, total, nil
}

// Update updates a role
func (r *RoleRepo) Update(ctx context.Context, role *domain.Role) error {
	query := `
		UPDATE roles SET
			code = $1, name = $2, description = $3, status = $4, updated_by = $5, updated_at = $6
		WHERE id = $7 AND is_system_defined = false
	`

	_, err := r.db.Exec(ctx, query,
		role.Code, role.Name, role.Description, role.Status,
		role.UpdatedBy, role.UpdatedAt, role.ID,
	)
	return err
}

// Delete deletes a role
func (r *RoleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM roles WHERE id = $1 AND is_system_defined = false`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// GetPermissionsByRoleID gets permissions for a role
func (r *RoleRepo) GetPermissionsByRoleID(ctx context.Context, roleID uuid.UUID) ([]domain.Permission, error) {
	query := `
		SELECT p.id, p.resource, p.action, p.description, p.is_system_defined, p.created_at
		FROM permissions p
		INNER JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1
	`

	rows, err := r.db.Query(ctx, query, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []domain.Permission
	for rows.Next() {
		var perm domain.Permission
		err := rows.Scan(
			&perm.ID, &perm.Resource, &perm.Action, &perm.Description,
			&perm.IsSystemDefined, &perm.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		perms = append(perms, perm)
	}

	return perms, nil
}

// AssignPermission assigns a permission to a role
func (r *RoleRepo) AssignPermission(ctx context.Context, roleID, permissionID uuid.UUID, assignedBy *uuid.UUID) error {
	query := `
		INSERT INTO role_permissions (role_id, permission_id, assigned_at, assigned_by)
		VALUES ($1, $2, NOW(), $3)
		ON CONFLICT (role_id, permission_id) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query, roleID, permissionID, assignedBy)
	return err
}

// RemovePermission removes a permission from a role
func (r *RoleRepo) RemovePermission(ctx context.Context, roleID, permissionID uuid.UUID) error {
	query := `DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = $2`
	_, err := r.db.Exec(ctx, query, roleID, permissionID)
	return err
}

// GetAllPermissions gets all permissions
func (r *RoleRepo) GetAllPermissions(ctx context.Context) ([]domain.Permission, error) {
	query := `
		SELECT id, resource, action, description, is_system_defined, created_at
		FROM permissions ORDER BY resource, action
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []domain.Permission
	for rows.Next() {
		var perm domain.Permission
		err := rows.Scan(
			&perm.ID, &perm.Resource, &perm.Action, &perm.Description,
			&perm.IsSystemDefined, &perm.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		perms = append(perms, perm)
	}

	return perms, nil
}

// CreatePermission creates a new permission
func (r *RoleRepo) CreatePermission(ctx context.Context, perm *domain.Permission) error {
	query := `
		INSERT INTO permissions (id, resource, action, description, is_system_defined, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.Exec(ctx, query,
		perm.ID, perm.Resource, perm.Action, perm.Description,
		perm.IsSystemDefined, perm.CreatedAt,
	)
	return err
}

// GetRolesByUserID gets roles for a user
func (r *RoleRepo) GetRolesByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Role, error) {
	query := `
		SELECT r.id, r.tenant_id, r.code, r.name, r.description, r.is_system_defined, r.status,
			r.created_by, r.updated_by, r.created_at, r.updated_at
		FROM roles r
		INNER JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1 AND r.status = $2
		AND (ur.expires_at IS NULL OR ur.expires_at > NOW())
	`

	rows, err := r.db.Query(ctx, query, userID, domain.RoleStatusActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []domain.Role
	for rows.Next() {
		var role domain.Role
		err := rows.Scan(
			&role.ID, &role.TenantID, &role.Code, &role.Name, &role.Description,
			&role.IsSystemDefined, &role.Status, &role.CreatedBy, &role.UpdatedBy,
			&role.CreatedAt, &role.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// AssignRoleToUser assigns a role to a user
func (r *RoleRepo) AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID, assignedBy *uuid.UUID, expiresAt *interface{}) error {
	query := `
		INSERT INTO user_roles (user_id, role_id, assigned_at, assigned_by, expires_at)
		VALUES ($1, $2, NOW(), $3, $4)
		ON CONFLICT (user_id, role_id) DO UPDATE SET
			assigned_at = EXCLUDED.assigned_at,
			assigned_by = EXCLUDED.assigned_by,
			expires_at = EXCLUDED.expires_at
	`

	_, err := r.db.Exec(ctx, query, userID, roleID, assignedBy, nil)
	return err
}

// RemoveRoleFromUser removes a role from a user
func (r *RoleRepo) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	query := `DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`
	_, err := r.db.Exec(ctx, query, userID, roleID)
	return err
}

// GetUserPermissions gets all permissions for a user (through roles)
func (r *RoleRepo) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]domain.Permission, error) {
	query := `
		SELECT DISTINCT p.id, p.resource, p.action, p.description, p.is_system_defined, p.created_at
		FROM permissions p
		INNER JOIN role_permissions rp ON p.id = rp.permission_id
		INNER JOIN user_roles ur ON rp.role_id = ur.role_id
		WHERE ur.user_id = $1
		AND (ur.expires_at IS NULL OR ur.expires_at > NOW())
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []domain.Permission
	for rows.Next() {
		var perm domain.Permission
		err := rows.Scan(
			&perm.ID, &perm.Resource, &perm.Action, &perm.Description,
			&perm.IsSystemDefined, &perm.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		perms = append(perms, perm)
	}

	return perms, nil
}
