package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/anto1290/qlxion-monorepo/pkg/response"
	"github.com/anto1290/qlxion-monorepo/services/auth-service/internal/domain"
	"github.com/anto1290/qlxion-monorepo/services/auth-service/internal/usecase"
	"github.com/google/uuid"
)

// UserHandler handles user management HTTP requests
type UserHandler struct {
	userUC *usecase.UserUsecase
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userUC *usecase.UserUsecase) *UserHandler {
	return &UserHandler{userUC: userUC}
}

// RegisterRoutes registers user routes
func (h *UserHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/users", h.ListUsers)
	mux.HandleFunc("POST /v1/users", h.CreateUser)
	mux.HandleFunc("GET /v1/users/{id}", h.GetUser)
	mux.HandleFunc("PUT /v1/users/{id}", h.UpdateUser)
	mux.HandleFunc("DELETE /v1/users/{id}", h.DeleteUser)
	mux.HandleFunc("GET /v1/users/{id}/roles", h.GetUserRoles)
	mux.HandleFunc("POST /v1/users/{id}/roles", h.AssignRole)
	mux.HandleFunc("DELETE /v1/users/{id}/roles/{roleId}", h.RemoveRole)
}

// ListUsers lists all users
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	filter := domain.UserFilter{}

	if tenantID := r.URL.Query().Get("tenant_id"); tenantID != "" {
		if id, err := uuid.Parse(tenantID); err == nil {
			filter.TenantID = &id
		}
	}

	if status := r.URL.Query().Get("status"); status != "" {
		if s, err := strconv.Atoi(status); err == nil {
			us := domain.UserStatus(s)
			filter.Status = &us
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

	users, total, err := h.userUC.ListUsers(ctx, filter)
	if err != nil {
		if appErr, ok := err.(*response.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, response.New(response.ErrInternal, "Failed to list users").WithError(err))
		}
		return
	}

	meta := response.Paginated(filter.Offset/filter.Limit+1, filter.Limit, total)
	response.JSON(w, http.StatusOK, response.SuccessWithMeta(users, meta))
}

// CreateUser creates a new user
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req usecase.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, response.New(response.ErrBadRequest, "Invalid request body").WithError(err))
		return
	}

	ctx := r.Context()
	user, err := h.userUC.CreateUser(ctx, req)
	if err != nil {
		if appErr, ok := err.(*response.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, response.New(response.ErrInternal, "Failed to create user").WithError(err))
		}
		return
	}

	response.JSONCreated(w, user)
}

// GetUser gets a user by ID
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.JSONError(w, response.New(response.ErrValidation, "Invalid user ID").WithError(err))
		return
	}

	ctx := r.Context()
	user, err := h.userUC.GetUser(ctx, id)
	if err != nil {
		if appErr, ok := err.(*response.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, response.New(response.ErrInternal, "Failed to get user").WithError(err))
		}
		return
	}

	response.JSONSuccess(w, user)
}

// UpdateUser updates a user
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.JSONError(w, response.New(response.ErrValidation, "Invalid user ID").WithError(err))
		return
	}

	var req usecase.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, response.New(response.ErrBadRequest, "Invalid request body").WithError(err))
		return
	}

	ctx := r.Context()
	user, err := h.userUC.UpdateUser(ctx, id, req)
	if err != nil {
		if appErr, ok := err.(*response.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, response.New(response.ErrInternal, "Failed to update user").WithError(err))
		}
		return
	}

	response.JSONSuccess(w, user)
}

// DeleteUser soft deletes a user
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.JSONError(w, response.New(response.ErrValidation, "Invalid user ID").WithError(err))
		return
	}

	// Get deleted_by from context (current user)
	var deletedBy uuid.UUID

	ctx := r.Context()
	if err := h.userUC.DeleteUser(ctx, id, deletedBy); err != nil {
		if appErr, ok := err.(*response.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, response.New(response.ErrInternal, "Failed to delete user").WithError(err))
		}
		return
	}

	response.NoContent(w)
}

// GetUserRoles gets roles for a user
func (h *UserHandler) GetUserRoles(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.JSONError(w, response.New(response.ErrValidation, "Invalid user ID").WithError(err))
		return
	}

	ctx := r.Context()
	roles, err := h.userUC.GetUserRoles(ctx, id)
	if err != nil {
		if appErr, ok := err.(*response.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, response.New(response.ErrInternal, "Failed to get user roles").WithError(err))
		}
		return
	}

	response.JSONSuccess(w, roles)
}

// AssignRole assigns a role to a user
func (h *UserHandler) AssignRole(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.JSONError(w, response.New(response.ErrValidation, "Invalid user ID").WithError(err))
		return
	}

	var req struct {
		RoleID uuid.UUID `json:"role_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, response.New(response.ErrBadRequest, "Invalid request body").WithError(err))
		return
	}

	ctx := r.Context()
	if err := h.userUC.AssignRole(ctx, id, req.RoleID, nil); err != nil {
		if appErr, ok := err.(*response.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, response.New(response.ErrInternal, "Failed to assign role").WithError(err))
		}
		return
	}

	response.JSONSuccess(w, map[string]string{"message": "Role assigned successfully"})
}

// RemoveRole removes a role from a user
func (h *UserHandler) RemoveRole(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.JSONError(w, response.New(response.ErrValidation, "Invalid user ID").WithError(err))
		return
	}

	roleID, err := uuid.Parse(r.PathValue("roleId"))
	if err != nil {
		response.JSONError(w, response.New(response.ErrValidation, "Invalid role ID").WithError(err))
		return
	}

	ctx := r.Context()
	if err := h.userUC.RemoveRole(ctx, id, roleID); err != nil {
		if appErr, ok := err.(*response.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, response.New(response.ErrInternal, "Failed to remove role").WithError(err))
		}
		return
	}

	response.JSONSuccess(w, map[string]string{"message": "Role removed successfully"})
}
