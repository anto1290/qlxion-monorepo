package response

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/qlxion/qlxion-monorepo/pkg/errors"
)

// Status represents JSend status
type Status string

const (
	StatusSuccess Status = "success"
	StatusFail    Status = "fail"
	StatusError   Status = "error"
)

// Response represents a standardized API response following JSend specification
type Response struct {
	Status    Status      `json:"status"`
	Data      interface{} `json:"data,omitempty"`
	Message   string      `json:"message,omitempty"`
	Code      string      `json:"code,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Meta      *Meta       `json:"meta,omitempty"`
}

// Meta represents pagination metadata
type Meta struct {
	Page       int    `json:"page"`
	PerPage    int    `json:"per_page"`
	Total      int64  `json:"total"`
	TotalPages int    `json:"total_pages"`
	Sort       string `json:"sort,omitempty"`
	Order      string `json:"order,omitempty"`
}

// Success returns a success response
func Success(data interface{}) Response {
	return Response{
		Status:    StatusSuccess,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// SuccessWithMeta returns a success response with pagination metadata
func SuccessWithMeta(data interface{}, meta Meta) Response {
	return Response{
		Status:    StatusSuccess,
		Data:      data,
		Timestamp: time.Now(),
		Meta:      &meta,
	}
}

// Created returns a created response
func Created(data interface{}) Response {
	return Response{
		Status:    StatusSuccess,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// Fail returns a fail response (client error with data)
func Fail(data interface{}, message string) Response {
	return Response{
		Status:    StatusFail,
		Data:      data,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// Error returns an error response
func Error(err *errors.AppError) Response {
	return Response{
		Status:    StatusError,
		Message:   err.Message,
		Code:      string(err.Code),
		Timestamp: time.Now(),
	}
}

// ErrorWithDetail returns an error response with detail
func ErrorWithDetail(err *errors.AppError, detail string) Response {
	return Response{
		Status:    StatusError,
		Message:   err.Message,
		Code:      string(err.Code),
		Timestamp: time.Now(),
	}
}

// JSON writes a JSON response
func JSON(w http.ResponseWriter, statusCode int, resp Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}

// JSONSuccess writes a success JSON response
func JSONSuccess(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, Success(data))
}

// JSONCreated writes a created JSON response
func JSONCreated(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusCreated, Created(data))
}

// JSONFail writes a fail JSON response
func JSONFail(w http.ResponseWriter, statusCode int, data interface{}, message string) {
	JSON(w, statusCode, Fail(data, message))
}

// JSONError writes an error JSON response from AppError
func JSONError(w http.ResponseWriter, err *errors.AppError) {
	JSON(w, err.Status, Error(err))
}

// NoContent writes a 204 No Content response
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Paginated creates pagination metadata
func Paginated(page, perPage int, total int64) Meta {
	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}
	if totalPages == 0 {
		totalPages = 1
	}

	return Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}
}
