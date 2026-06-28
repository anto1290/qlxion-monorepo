package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	appErrors "github.com/anto1290/qlxion-monorepo/pkg/errors"
	"github.com/anto1290/qlxion-monorepo/pkg/response"
	"github.com/anto1290/qlxion-monorepo/services/auth-service/internal/domain"
	"github.com/anto1290/qlxion-monorepo/services/auth-service/internal/usecase"
	"github.com/google/uuid"
)

// RoleHandler handles role and permission HTTP requests
type RoleHandler struct {
	roleUC *usecase.RoleUsecase
}

// NewRoleHandler creates a new RoleHandler
func NewRoleHandler(roleUC *usecase.RoleUsecase) *RoleHandler {
	return &RoleHandler{roleUC: roleUC}
}

// RegisterRoutes registers role routes
func (h *RoleHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/roles", h.ListRoles)
	mux.HandleFunc("POST /v1/roles", h.CreateRole)
	mux.HandleFunc("GET /v1/roles/{id}", h.GetRole)
	mux.HandleFunc("PUT /v1/roles/{id}", h.UpdateRole)
	mux.HandleFunc("DELETE /v1/roles/{id}", h.DeleteRole)
	mux.HandleFunc("GET /v1/roles/{id}/permissions", h.GetRolePermissions)
	mux.HandleFunc("POST /v1/roles/{id}/permissions", h.AssignPermission)
	mux.HandleFunc("DELETE /v1/roles/{id}/permissions/{permId}", h.RemovePermission)
	mux.HandleFunc("GET /v1/permissions", h.ListPermissions)
	mux.HandleFunc("POST /v1/permissions", h.CreatePermission)
}

// ListRoles lists all roles
func (h *RoleHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	filter := domain.RoleFilter{}

	if tenantID := r.URL.Query().Get("tenant_id"); tenantID != "" {
		if id, err := uuid.Parse(tenantID); err == nil {
			filter.TenantID = &id
		}
	}

	if status := r.URL.Query().Get("status"); status != "" {
		if s, err := strconv.Atoi(status); err == nil {
			rs := domain.RoleStatus(s)
			filter.Status = &rs
		}
	}

	if search := r.URL.Query().Get("search"); search != "" {
		filter.Search = &search
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

	roles, total, err := h.roleUC.ListRoles(ctx, filter)
	if err != nil {
		if appErr, ok := err.(*appErrors.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, appErrors.New(appErrors.ErrInternal, "Failed to list roles").WithError(err))
		}
		return
	}

	meta := response.Paginated(filter.Offset/filter.Limit+1, filter.Limit, total)
	response.JSON(w, http.StatusOK, response.SuccessWithMeta(roles, meta))
}

// CreateRole creates a new role
func (h *RoleHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var req usecase.CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, appErrors.New(appErrors.ErrBadRequest, "Invalid request body").WithError(err))
		return
	}

	ctx := r.Context()
	role, err := h.roleUC.CreateRole(ctx, req)
	if err != nil {
		if appErr, ok := err.(*appErrors.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, appErrors.New(appErrors.ErrInternal, "Failed to create role").WithError(err))
		}
		return
	}

	response.JSONCreated(w, role)
}

// GetRole gets a role by ID
func (h *RoleHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.JSONError(w, appErrors.New(appErrors.ErrValidation, "Invalid role ID").WithError(err))
		return
	}

	ctx := r.Context()
	role, err := h.roleUC.GetRole(ctx, id)
	if err != nil {
		if appErr, ok := err.(*appErrors.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, appErrors.New(appErrors.ErrInternal, "Failed to get role").WithError(err))
		}
		return
	}

	response.JSONSuccess(w, role)
}

// UpdateRole updates a role
func (h *RoleHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.JSONError(w, appErrors.New(appErrors.ErrValidation, "Invalid role ID").WithError(err))
		return
	}

	var req usecase.UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, appErrors.New(appErrors.ErrBadRequest, "Invalid request body").WithError(err))
		return
	}

	ctx := r.Context()
	role, err := h.roleUC.UpdateRole(ctx, id, req)
	if err != nil {
		if appErr, ok := err.(*appErrors.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, appErrors.New(appErrors.ErrInternal, "Failed to update role").WithError(err))
		}
		return
	}

	response.JSONSuccess(w, role)
}

// DeleteRole deletes a role
func (h *RoleHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.JSONError(w, appErrors.New(appErrors.ErrValidation, "Invalid role ID").WithError(err))
		return
	}

	ctx := r.Context()
	if err := h.roleUC.DeleteRole(ctx, id); err != nil {
		if appErr, ok := err.(*appErrors.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, appErrors.New(appErrors.ErrInternal, "Failed to delete role").WithError(err))
		}
		return
	}

	response.NoContent(w)
}

// GetRolePermissions gets permissions for a role
func (h *RoleHandler) GetRolePermissions(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.JSONError(w, appErrors.New(appErrors.ErrValidation, "Invalid role ID").WithError(err))
		return
	}

	ctx := r.Context()
	permissions, err := h.roleUC.GetRolePermissions(ctx, id)
	if err != nil {
		if appErr, ok := err.(*appErrors.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, appErrors.New(appErrors.ErrInternal, "Failed to get role permissions").WithError(err))
		}
		return
	}

	response.JSONSuccess(w, permissions)
}

// AssignPermission assigns a permission to a role
func (h *RoleHandler) AssignPermission(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.JSONError(w, appErrors.New(appErrors.ErrValidation, "Invalid role ID").WithError(err))
		return
	}

	var req struct {
		PermissionID uuid.UUID `json:"permission_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, appErrors.New(appErrors.ErrBadRequest, "Invalid request body").WithError(err))
		return
	}

	ctx := r.Context()
	if err := h.roleUC.AssignPermission(ctx, id, req.PermissionID, nil); err != nil {
		if appErr, ok := err.(*appErrors.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, appErrors.New(appErrors.ErrInternal, "Failed to assign permission").WithError(err))
		}
		return
	}

	response.JSONSuccess(w, map[string]string{"message": "Permission assigned successfully"})
}

// RemovePermission removes a permission from a role
func (h *RoleHandler) RemovePermission(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.JSONError(w, appErrors.New(appErrors.ErrValidation, "Invalid role ID").WithError(err))
		return
	}

	permID, err := uuid.Parse(r.PathValue("permId"))
	if err != nil {
		response.JSONError(w, appErrors.New(appErrors.ErrValidation, "Invalid permission ID").WithError(err))
		return
	}

	ctx := r.Context()
	if err := h.roleUC.RemovePermission(ctx, id, permID); err != nil {
		if appErr, ok := err.(*appErrors.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, appErrors.New(appErrors.ErrInternal, "Failed to remove permission").WithError(err))
		}
		return
	}

	response.JSONSuccess(w, map[string]string{"message": "Permission removed successfully"})
}

// ListPermissions lists all permissions
func (h *RoleHandler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	permissions, err := h.roleUC.GetPermissions(ctx)
	if err != nil {
		if appErr, ok := err.(*appErrors.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, appErrors.New(appErrors.ErrInternal, "Failed to list permissions").WithError(err))
		}
		return
	}

	response.JSONSuccess(w, permissions)
}

// CreatePermission creates a new permission
func (h *RoleHandler) CreatePermission(w http.ResponseWriter, r *http.Request) {
	var req usecase.CreatePermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, appErrors.New(appErrors.ErrBadRequest, "Invalid request body").WithError(err))
		return
	}

	ctx := r.Context()
	permission, err := h.roleUC.CreatePermission(ctx, req)
	if err != nil {
		if appErr, ok := err.(*appErrors.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, appErrors.New(appErrors.ErrInternal, "Failed to create permission").WithError(err))
		}
		return
	}

	response.JSONCreated(w, permission)
}
