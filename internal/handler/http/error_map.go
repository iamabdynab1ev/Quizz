package http

import (
	"errors"
	nethttp "net/http"

	"lms-arvand-backend/internal/domain"
)

func writeMappedError(w nethttp.ResponseWriter, err error) int {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		writeAppError(w, appErr)
		return appErr.Status
	}

	switch {
	case errors.Is(err, domain.ErrValidation):
		writeError(w, nethttp.StatusBadRequest, "validation_error", "Проверьте данные запроса")
		return nethttp.StatusBadRequest
	case errors.Is(err, domain.ErrUnauthorized):
		writeError(w, nethttp.StatusUnauthorized, "unauthorized", "Требуется авторизация")
		return nethttp.StatusUnauthorized
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, nethttp.StatusForbidden, "forbidden", "Недостаточно прав")
		return nethttp.StatusForbidden
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, nethttp.StatusNotFound, "not_found", "Запись не найдена")
		return nethttp.StatusNotFound
	case errors.Is(err, domain.ErrConflict):
		writeError(w, nethttp.StatusConflict, "conflict", "Запись конфликтует с уже существующими данными")
		return nethttp.StatusConflict
	case errors.Is(err, domain.ErrUnavailable):
		writeError(w, nethttp.StatusServiceUnavailable, "service_unavailable", "Сервис временно недоступен")
		return nethttp.StatusServiceUnavailable
	case errors.Is(err, domain.ErrTooMany):
		writeError(w, nethttp.StatusTooManyRequests, "too_many_requests", "Слишком много запросов. Попробуйте позже")
		return nethttp.StatusTooManyRequests
	default:
		writeError(w, nethttp.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		return nethttp.StatusInternalServerError
	}
}
