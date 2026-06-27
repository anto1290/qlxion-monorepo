package usecase

import (
	"context"

	"github.com/qlxion/qlxion-monorepo/services/auth-service/internal/domain"
	"github.com/qlxion/qlxion-monorepo/services/auth-service/internal/repository"
)

// AuditUsecase handles audit log business logic
type AuditUsecase struct {
	auditRepo repository.AuditRepository
}

// NewAuditUsecase creates a new AuditUsecase
func NewAuditUsecase(auditRepo repository.AuditRepository) *AuditUsecase {
	return &AuditUsecase{auditRepo: auditRepo}
}

// ListAuditLogs lists audit logs with filter
func (u *AuditUsecase) ListAuditLogs(ctx context.Context, filter domain.AuditLogFilter) ([]domain.AuditLog, int64, error) {
	return u.auditRepo.List(ctx, filter)
}
