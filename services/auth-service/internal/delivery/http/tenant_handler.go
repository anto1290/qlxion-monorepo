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

// TenantHandler handles tenant management HTTP requests
type TenantHandler struct {
	tenantUC *usecase.TenantUsecase
}

// NewTenantHandler creates a new TenantHandler
func NewTenantHandler(tenantUC *usecase.TenantUsecase) *TenantHandler {
	return &TenantHandler{tenantUC: tenantUC}
}

// RegisterRoutes registers tenant routes
func (h *TenantHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/tenants", h.ListTenants)
	mux.HandleFunc("POST /v1/tenants", h.CreateTenant)
	mux.HandleFunc("GET /v1/tenants/{id}", h.GetTenant)
	mux.HandleFunc("PUT /v1/tenants/{id}", h.UpdateTenant)
	mux.HandleFunc("DELETE /v1/tenants/{id}", h.DeleteTenant)
}

// ListTenants lists all tenants
func (h *TenantHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	filter := domain.TenantFilter{}

	if status := r.URL.Query().Get("status"); status != "" {
		if s, err := strconv.Atoi(status); err == nil {
			ts := domain.TenantStatus(s)
			filter.Status = &ts
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

	tenants, total, err := h.tenantUC.ListTenants(ctx, filter)
	if err != nil {
		if appErr, ok := err.(*response.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, response.New(response.ErrInternal, "Failed to list tenants").WithError(err))
		}
		return
	}

	meta := response.Paginated(filter.Offset/filter.Limit+1, filter.Limit, total)
	response.JSON(w, http.StatusOK, response.SuccessWithMeta(tenants, meta))
}

// CreateTenant creates a new tenant
func (h *TenantHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	var req usecase.CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, response.New(response.ErrBadRequest, "Invalid request body").WithError(err))
		return
	}

	ctx := r.Context()
	tenant, err := h.tenantUC.CreateTenant(ctx, req)
	if err != nil {
		if appErr, ok := err.(*response.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, response.New(response.ErrInternal, "Failed to create tenant").WithError(err))
		}
		return
	}

	response.JSONCreated(w, tenant)
}

// GetTenant gets a tenant by ID
func (h *TenantHandler) GetTenant(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.JSONError(w, response.New(response.ErrValidation, "Invalid tenant ID").WithError(err))
		return
	}

	ctx := r.Context()
	tenant, err := h.tenantUC.GetTenant(ctx, id)
	if err != nil {
		if appErr, ok := err.(*response.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, response.New(response.ErrInternal, "Failed to get tenant").WithError(err))
		}
		return
	}

	response.JSONSuccess(w, tenant)
}

// UpdateTenant updates a tenant
func (h *TenantHandler) UpdateTenant(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.JSONError(w, response.New(response.ErrValidation, "Invalid tenant ID").WithError(err))
		return
	}

	var req usecase.UpdateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, response.New(response.ErrBadRequest, "Invalid request body").WithError(err))
		return
	}

	ctx := r.Context()
	tenant, err := h.tenantUC.UpdateTenant(ctx, id, req)
	if err != nil {
		if appErr, ok := err.(*response.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, response.New(response.ErrInternal, "Failed to update tenant").WithError(err))
		}
		return
	}

	response.JSONSuccess(w, tenant)
}

// DeleteTenant soft deletes a tenant
func (h *TenantHandler) DeleteTenant(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.JSONError(w, response.New(response.ErrValidation, "Invalid tenant ID").WithError(err))
		return
	}

	var deletedBy uuid.UUID

	ctx := r.Context()
	if err := h.tenantUC.DeleteTenant(ctx, id, deletedBy); err != nil {
		if appErr, ok := err.(*response.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, response.New(response.ErrInternal, "Failed to delete tenant").WithError(err))
		}
		return
	}

	response.NoContent(w)
}
