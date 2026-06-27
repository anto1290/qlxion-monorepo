package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ErrorCode represents a standardized error code
type ErrorCode string

const (
	// General errors
	ErrInternal       ErrorCode = "INTERNAL_ERROR"
	ErrNotFound       ErrorCode = "NOT_FOUND"
	ErrBadRequest     ErrorCode = "BAD_REQUEST"
	ErrUnauthorized   ErrorCode = "UNAUTHORIZED"
	ErrForbidden      ErrorCode = "FORBIDDEN"
	ErrConflict       ErrorCode = "CONFLICT"
	ErrValidation     ErrorCode = "VALIDATION_ERROR"
	ErrTooManyRequest ErrorCode = "TOO_MANY_REQUESTS"
	ErrServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	
	// Auth specific errors
	ErrInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
	ErrTokenExpired       ErrorCode = "TOKEN_EXPIRED"
	ErrTokenInvalid       ErrorCode = "TOKEN_INVALID"
	ErrSessionRevoked     ErrorCode = "SESSION_REVOKED"
	ErrUserInactive       ErrorCode = "USER_INACTIVE"
	ErrTenantInactive     ErrorCode = "TENANT_INACTIVE"
)

// HTTPStatus maps error codes to HTTP status codes
var httpStatusMap = map[ErrorCode]int{
	ErrInternal:           http.StatusInternalServerError,
	ErrNotFound:           http.StatusNotFound,
	ErrBadRequest:         http.StatusBadRequest,
	ErrUnauthorized:       http.StatusUnauthorized,
	ErrForbidden:          http.StatusForbidden,
	ErrConflict:           http.StatusConflict,
	ErrValidation:         http.StatusUnprocessableEntity,
	ErrTooManyRequest:     http.StatusTooManyRequests,
	ErrServiceUnavailable: http.StatusServiceUnavailable,
	ErrInvalidCredentials: http.StatusUnauthorized,
	ErrTokenExpired:       http.StatusUnauthorized,
	ErrTokenInvalid:       http.StatusUnauthorized,
	ErrSessionRevoked:     http.StatusUnauthorized,
	ErrUserInactive:       http.StatusForbidden,
	ErrTenantInactive:     http.StatusForbidden,
}

// AppError represents a standardized application error
type AppError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Detail  string    `json:"detail,omitempty"`
	Status  int       `json:"status"`
	Err     error     `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error
func (e *AppError) Unwrap() error {
	return e.Err
}

// MarshalJSON implements json.Marshaler
func (e *AppError) MarshalJSON() ([]byte, error) {
	type errorJSON struct {
		Code    ErrorCode `json:"code"`
		Message string    `json:"message"`
		Detail  string    `json:"detail,omitempty"`
		Status  int       `json:"status"`
	}
	return json.Marshal(errorJSON{
		Code:    e.Code,
		Message: e.Message,
		Detail:  e.Detail,
		Status:  e.Status,
	})
}

// New creates a new AppError
func New(code ErrorCode, message string) *AppError {
	status, ok := httpStatusMap[code]
	if !ok {
		status = http.StatusInternalServerError
	}
	return &AppError{
		Code:    code,
		Message: message,
		Status:  status,
	}
}

// Wrap wraps an existing error with a code
func Wrap(code ErrorCode, message string, err error) *AppError {
	appErr := New(code, message)
	appErr.Err = err
	if err != nil {
		appErr.Detail = err.Error()
	}
	return appErr
}

// WithDetail adds detail to an error
func (e *AppError) WithDetail(detail string) *AppError {
	e.Detail = detail
	return e
}

// WithError wraps an underlying error
func (e *AppError) WithError(err error) *AppError {
	e.Err = err
	if err != nil {
		e.Detail = err.Error()
	}
	return e
}

// IsNotFound checks if error is a not found error
func IsNotFound(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == ErrNotFound
	}
	return false
}

// IsUnauthorized checks if error is an unauthorized error
func IsUnauthorized(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == ErrUnauthorized
	}
	return false
}

// GetStatusCode extracts HTTP status code from error
func GetStatusCode(err error) int {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Status
	}
	return http.StatusInternalServerError
}
