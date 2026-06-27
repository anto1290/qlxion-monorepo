package http

import (
	"net/http"
	"strconv"

	"github.com/anto1290/qlxion-monorepo/pkg/auth"
	"github.com/anto1290/qlxion-monorepo/pkg/response"
	"github.com/anto1290/qlxion-monorepo/services/auth-service/internal/domain"
	"github.com/anto1290/qlxion-monorepo/services/auth-service/internal/usecase"
	"github.com/google/uuid"
)

// SessionHandler handles session management HTTP requests
type SessionHandler struct {
	sessionUC *usecase.SessionUsecase
}

// NewSessionHandler creates a new SessionHandler
func NewSessionHandler(sessionUC *usecase.SessionUsecase) *SessionHandler {
	return &SessionHandler{sessionUC: sessionUC}
}

// RegisterRoutes registers session routes
func (h *SessionHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/sessions", h.ListSessions)
	mux.HandleFunc("POST /v1/sessions/{id}/revoke", h.RevokeSession)
}

// ListSessions lists sessions for the authenticated user
func (h *SessionHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ExtractClaimsFromContext(r.Context())
	if !ok {
		response.JSONError(w, response.New(response.ErrUnauthorized, "Not authenticated"))
		return
	}

	ctx := r.Context()

	filter := domain.SessionFilter{}

	if deviceType := r.URL.Query().Get("device_type"); deviceType != "" {
		filter.DeviceType = &deviceType
	}

	if status := r.URL.Query().Get("status"); status == "active" {
		active := false
		filter.IsRevoked = &active
	} else if status == "revoked" {
		revoked := true
		filter.IsRevoked = &revoked
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		filter.Limit, _ = strconv.Atoi(limit)
	}
	if filter.Limit == 0 {
		filter.Limit = 20
	}

	if offset := r.URL.Query().Get("offset"); offset != "" {
		filter.Offset, _ = strconv.Atoi(offset)
	}

	sessions, total, err := h.sessionUC.ListSessions(ctx, claims.UserID, filter)
	if err != nil {
		if appErr, ok := err.(*response.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, response.New(response.ErrInternal, "Failed to list sessions").WithError(err))
		}
		return
	}

	meta := response.Paginated(filter.Offset/filter.Limit+1, filter.Limit, total)
	response.JSON(w, http.StatusOK, response.SuccessWithMeta(sessions, meta))
}

// RevokeSession revokes a session
func (h *SessionHandler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.JSONError(w, response.New(response.ErrValidation, "Invalid session ID").WithError(err))
		return
	}

	ctx := r.Context()
	if err := h.sessionUC.RevokeSession(ctx, id); err != nil {
		if appErr, ok := err.(*response.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, response.New(response.ErrInternal, "Failed to revoke session").WithError(err))
		}
		return
	}

	response.JSONSuccess(w, map[string]string{"message": "Session revoked successfully"})
}

// AuditHandler handles audit log HTTP requests
type AuditHandler struct {
	auditUC *usecase.AuditUsecase
}

// NewAuditHandler creates a new AuditHandler
func NewAuditHandler(auditUC *usecase.AuditUsecase) *AuditHandler {
	return &AuditHandler{auditUC: auditUC}
}

// RegisterRoutes registers audit log routes
func (h *AuditHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/audit-logs", h.ListAuditLogs)
}

// ListAuditLogs lists audit logs
func (h *AuditHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	filter := domain.AuditLogFilter{}

	if tenantID := r.URL.Query().Get("tenant_id"); tenantID != "" {
		if id, err := uuid.Parse(tenantID); err == nil {
			filter.TenantID = &id
		}
	}

	if userID := r.URL.Query().Get("user_id"); userID != "" {
		if id, err := uuid.Parse(userID); err == nil {
			filter.UserID = &id
		}
	}

	if action := r.URL.Query().Get("action"); action != "" {
		a := domain.AuditAction(action)
		filter.Action = &a
	}

	if resource := r.URL.Query().Get("resource"); resource != "" {
		filter.Resource = &resource
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		filter.Limit, _ = strconv.Atoi(limit)
	}
	if filter.Limit == 0 {
		filter.Limit = 50
	}

	if offset := r.URL.Query().Get("offset"); offset != "" {
		filter.Offset, _ = strconv.Atoi(offset)
	}

	logs, total, err := h.auditUC.ListAuditLogs(ctx, filter)
	if err != nil {
		if appErr, ok := err.(*response.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, response.New(response.ErrInternal, "Failed to list audit logs").WithError(err))
		}
		return
	}

	meta := response.Paginated(filter.Offset/filter.Limit+1, filter.Limit, total)
	response.JSON(w, http.StatusOK, response.SuccessWithMeta(logs, meta))
}
