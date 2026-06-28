package http

import (
	"encoding/json"
	"net/http"

	"github.com/anto1290/qlxion-monorepo/pkg/auth"
	appErrors "github.com/anto1290/qlxion-monorepo/pkg/errors"
	"github.com/anto1290/qlxion-monorepo/pkg/response"
	"github.com/anto1290/qlxion-monorepo/services/auth-service/internal/domain"
	"github.com/anto1290/qlxion-monorepo/services/auth-service/internal/usecase"
)

// AuthHandler handles authentication HTTP requests
type AuthHandler struct {
	authUC *usecase.AuthUsecase
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(authUC *usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{authUC: authUC}
}

// RegisterRoutes registers auth routes
func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/auth/login", h.Login)
	mux.HandleFunc("POST /v1/auth/register", h.Register)
	mux.HandleFunc("POST /v1/auth/refresh", h.RefreshToken)
	mux.HandleFunc("POST /v1/auth/logout", h.Logout)
	mux.HandleFunc("GET /v1/auth/me", h.GetMe)
	mux.HandleFunc("POST /v1/auth/password/change", h.ChangePassword)
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, appErrors.New(appErrors.ErrBadRequest, "Invalid request body").WithError(err))
		return
	}

	// Get client info from request
	deviceName := r.Header.Get("X-Device-Name")
	deviceType := r.Header.Get("X-Device-Type")
	if deviceType == "" {
		deviceType = "web"
	}
	req.DeviceName = &deviceName
	req.DeviceType = &deviceType

	ctx := r.Context()
	result, err := h.authUC.Login(ctx, req)
	if err != nil {
		if appErr, ok := err.(*appErrors.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, appErrors.New(appErrors.ErrInternal, "Login failed").WithError(err))
		}
		return
	}

	response.JSONSuccess(w, result)
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req domain.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, appErrors.New(appErrors.ErrBadRequest, "Invalid request body").WithError(err))
		return
	}

	ctx := r.Context()
	result, err := h.authUC.Register(ctx, req)
	if err != nil {
		if appErr, ok := err.(*appErrors.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, appErrors.New(appErrors.ErrInternal, "Registration failed").WithError(err))
		}
		return
	}

	response.JSONCreated(w, result)
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req domain.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, appErrors.New(appErrors.ErrBadRequest, "Invalid request body").WithError(err))
		return
	}

	ctx := r.Context()
	tokenPair, err := h.authUC.RefreshToken(ctx, req)
	if err != nil {
		if appErr, ok := err.(*appErrors.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, appErrors.New(appErrors.ErrInternal, "Token refresh failed").WithError(err))
		}
		return
	}

	response.JSONSuccess(w, tokenPair)
}

// Logout handles user logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Get session ID from context (set by auth middleware)
	claims, ok := auth.ExtractClaimsFromContext(r.Context())
	if !ok {
		response.JSONError(w, appErrors.New(appErrors.ErrUnauthorized, "Not authenticated"))
		return
	}

	ctx := r.Context()
	if err := h.authUC.Logout(ctx, claims.SessionID); err != nil {
		if appErr, ok := err.(*appErrors.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, appErrors.New(appErrors.ErrInternal, "Logout failed").WithError(err))
		}
		return
	}

	response.NoContent(w)
}

// GetMe gets current user info
func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ExtractClaimsFromContext(r.Context())
	if !ok {
		response.JSONError(w, appErrors.New(appErrors.ErrUnauthorized, "Not authenticated"))
		return
	}

	ctx := r.Context()
	user, err := h.authUC.GetMe(ctx, claims.UserID)
	if err != nil {
		if appErr, ok := err.(*appErrors.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, appErrors.New(appErrors.ErrInternal, "Failed to get user info").WithError(err))
		}
		return
	}

	response.JSONSuccess(w, user)
}

// ChangePassword handles password change
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ExtractClaimsFromContext(r.Context())
	if !ok {
		response.JSONError(w, appErrors.New(appErrors.ErrUnauthorized, "Not authenticated"))
		return
	}

	var req domain.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, appErrors.New(appErrors.ErrBadRequest, "Invalid request body").WithError(err))
		return
	}

	ctx := r.Context()
	if err := h.authUC.ChangePassword(ctx, claims.UserID, req); err != nil {
		if appErr, ok := err.(*appErrors.AppError); ok {
			response.JSONError(w, appErr)
		} else {
			response.JSONError(w, appErrors.New(appErrors.ErrInternal, "Password change failed").WithError(err))
		}
		return
	}

	response.JSONSuccess(w, map[string]string{"message": "Password changed successfully"})
}
