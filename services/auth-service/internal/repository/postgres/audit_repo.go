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

// AuditRepo implements AuditRepository
type AuditRepo struct {
	db *pgxpool.Pool
}

// NewAuditRepo creates a new AuditRepo
func NewAuditRepo(db *pgxpool.Pool) *AuditRepo {
	return &AuditRepo{db: db}
}

// Create creates a new audit log entry
func (r *AuditRepo) Create(ctx context.Context, log *domain.AuditLog) error {
	query := `
		INSERT INTO audit_logs (id, tenant_id, user_id, action, resource, resource_id, details, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.Exec(ctx, query,
		log.ID, log.TenantID, log.UserID, log.Action,
		log.Resource, log.ResourceID, log.Details,
		log.IPAddress, log.UserAgent, log.CreatedAt,
	)
	return err
}

// List lists audit logs with filter
func (r *AuditRepo) List(ctx context.Context, filter domain.AuditLogFilter) ([]domain.AuditLog, int64, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.TenantID != nil {
		conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
		args = append(args, *filter.TenantID)
		argIdx++
	}

	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIdx))
		args = append(args, *filter.UserID)
		argIdx++
	}

	if filter.Action != nil {
		conditions = append(conditions, fmt.Sprintf("action = $%d", argIdx))
		args = append(args, *filter.Action)
		argIdx++
	}

	if filter.Resource != nil {
		conditions = append(conditions, fmt.Sprintf("resource = $%d", argIdx))
		args = append(args, *filter.Resource)
		argIdx++
	}

	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *filter.StartDate)
		argIdx++
	}

	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *filter.EndDate)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int64
	countQuery := `SELECT COUNT(*) FROM audit_logs ` + whereClause
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
		limit = 50
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, user_id, action, resource, resource_id, details,
			ip_address, user_agent, created_at
		FROM audit_logs %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sort, order, argIdx, argIdx+1)

	args = append(args, limit, filter.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []domain.AuditLog
	for rows.Next() {
		var log domain.AuditLog
		err := rows.Scan(
			&log.ID, &log.TenantID, &log.UserID, &log.Action,
			&log.Resource, &log.ResourceID, &log.Details,
			&log.IPAddress, &log.UserAgent, &log.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		logs = append(logs, log)
	}

	return logs, total, nil
}

// GetByID gets audit log by ID
func (r *AuditRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.AuditLog, error) {
	query := `
		SELECT id, tenant_id, user_id, action, resource, resource_id, details,
			ip_address, user_agent, created_at
		FROM audit_logs WHERE id = $1
	`

	log := &domain.AuditLog{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&log.ID, &log.TenantID, &log.UserID, &log.Action,
		&log.Resource, &log.ResourceID, &log.Details,
		&log.IPAddress, &log.UserAgent, &log.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("audit log not found")
		}
		return nil, err
	}
	return log, nil
}
