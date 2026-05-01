package http

import (
	"errors"
	nethttp "net/http"

	"lms-arvand-backend/internal/domain"
)

func writeMappedError(w nethttp.ResponseWriter, err error) int {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		writeError(w, appErr.Status, appErr.Code, appErr.Message)
		return appErr.Status
	}

	switch {
	case errors.Is(err, domain.ErrValidation):
		writeError(w, nethttp.StatusBadRequest, "validation_error", "request validation failed")
		return nethttp.StatusBadRequest
	case errors.Is(err, domain.ErrUnauthorized):
		writeError(w, nethttp.StatusUnauthorized, "unauthorized", "authentication required")
		return nethttp.StatusUnauthorized
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, nethttp.StatusForbidden, "forbidden", "access denied")
		return nethttp.StatusForbidden
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, nethttp.StatusNotFound, "not_found", "resource not found")
		return nethttp.StatusNotFound
	case errors.Is(err, domain.ErrConflict):
		writeError(w, nethttp.StatusConflict, "conflict", "resource conflict")
		return nethttp.StatusConflict
	case errors.Is(err, domain.ErrUnavailable):
		writeError(w, nethttp.StatusServiceUnavailable, "service_unavailable", "service unavailable")
		return nethttp.StatusServiceUnavailable
	default:
		writeError(w, nethttp.StatusInternalServerError, "internal_error", "internal server error")
		return nethttp.StatusInternalServerError
	}
}
