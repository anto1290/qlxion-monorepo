package usecase

import (
	"context"

	"github.com/google/uuid"
	appErrors "github.com/qlxion/qlxion-monorepo/pkg/errors"
	"github.com/qlxion/qlxion-monorepo/services/auth-service/internal/domain"
	"github.com/qlxion/qlxion-monorepo/services/auth-service/internal/repository"
)

// SessionUsecase handles session management business logic
type SessionUsecase struct {
	sessionRepo repository.SessionRepository
	auditRepo   repository.AuditRepository
}

// NewSessionUsecase creates a new SessionUsecase
func NewSessionUsecase(
	sessionRepo repository.SessionRepository,
	auditRepo repository.AuditRepository,
) *SessionUsecase {
	return &SessionUsecase{
		sessionRepo: sessionRepo,
		auditRepo:   auditRepo,
	}
}

// ListSessions lists sessions for a user
func (u *SessionUsecase) ListSessions(ctx context.Context, userID uuid.UUID, filter domain.SessionFilter) ([]domain.Session, int64, error) {
	return u.sessionRepo.GetByUserID(ctx, userID, filter)
}

// RevokeSession revokes a specific session
func (u *SessionUsecase) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	session, err := u.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return appErrors.Wrap(appErrors.ErrNotFound, "Session not found", err)
	}

	if session.IsRevoked() {
		return appErrors.New(appErrors.ErrBadRequest, "Session already revoked")
	}

	if err := u.sessionRepo.Revoke(ctx, sessionID); err != nil {
		return appErrors.Wrap(appErrors.ErrInternal, "Failed to revoke session", err)
	}

	// Log audit
	log := &domain.AuditLog{
		ID:         uuid.New(),
		UserID:     &session.UserID,
		Action:     domain.AuditActionSessionRevoke,
		ResourceID: &sessionID,
		CreatedAt:  session.CreatedAt,
	}
	u.auditRepo.Create(ctx, log)

	return nil
}

// RevokeAllOtherSessions revokes all sessions except the current one
func (u *SessionUsecase) RevokeAllOtherSessions(ctx context.Context, userID, currentSessionID uuid.UUID) error {
	if err := u.sessionRepo.RevokeAllUserSessions(ctx, userID, &currentSessionID); err != nil {
		return appErrors.Wrap(appErrors.ErrInternal, "Failed to revoke sessions", err)
	}
	return nil
}

// GetSession gets a session by ID
func (u *SessionUsecase) GetSession(ctx context.Context, sessionID uuid.UUID) (*domain.Session, error) {
	session, err := u.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, appErrors.Wrap(appErrors.ErrNotFound, "Session not found", err)
	}
	return session, nil
}

// UpdateSessionActivity updates session last activity
func (u *SessionUsecase) UpdateSessionActivity(ctx context.Context, sessionID uuid.UUID) error {
	if err := u.sessionRepo.UpdateActivity(ctx, sessionID); err != nil {
		return appErrors.Wrap(appErrors.ErrInternal, "Failed to update session activity", err)
	}
	return nil
}
