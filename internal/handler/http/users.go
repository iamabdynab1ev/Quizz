package http

import (
	"context"
	"fmt"
	"log/slog"
	nethttp "net/http"
	"strconv"

	"lms-arvand-backend/internal/domain"

	"github.com/go-chi/chi/v5"
)

type userUseCase interface {
	Create(ctx context.Context, params domain.CreateUserParams) (domain.User, error)
	GetByID(ctx context.Context, userID string) (domain.User, error)
	List(ctx context.Context, filter domain.UserListFilter) ([]domain.User, int, error)
	Update(ctx context.Context, params domain.UpdateUserParams) (domain.User, error)
	Deactivate(ctx context.Context, userID string) error
}

type UsersHandler struct {
	logger  *slog.Logger
	useCase userUseCase
}

func NewUsersHandler(logger *slog.Logger, useCase userUseCase) *UsersHandler {
	return &UsersHandler{
		logger:  logger,
		useCase: useCase,
	}
}

func (h *UsersHandler) CreateUser(w nethttp.ResponseWriter, r *nethttp.Request) {
	var params domain.CreateUserParams
	if err := decodeJSON(w, r, &params, 1<<20); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}

	user, err := h.useCase.Create(r.Context(), params)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "create user failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusCreated, user); err != nil {
		h.logger.ErrorContext(r.Context(), "create user response failed", slog.String("error", err.Error()))
	}
}

func (h *UsersHandler) GetUserByID(w nethttp.ResponseWriter, r *nethttp.Request) {
	user, err := h.useCase.GetByID(r.Context(), chi.URLParam(r, "userID"))
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "get user failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, user); err != nil {
		h.logger.ErrorContext(r.Context(), "get user response failed", slog.String("error", err.Error()))
	}
}

func (h *UsersHandler) ListUsers(w nethttp.ResponseWriter, r *nethttp.Request) {
	filter, err := h.parseUserListFilter(r)
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	users, total, err := h.useCase.List(r.Context(), filter)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "list users failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writePagedJSON(w, nethttp.StatusOK, users, total, filter.Limit, filter.Offset); err != nil {
		h.logger.ErrorContext(r.Context(), "list users response failed", slog.String("error", err.Error()))
	}
}

func (h *UsersHandler) UpdateUser(w nethttp.ResponseWriter, r *nethttp.Request) {
	var params domain.UpdateUserParams
	if err := decodeJSON(w, r, &params, 1<<20); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}

	params.ID = chi.URLParam(r, "userID")

	user, err := h.useCase.Update(r.Context(), params)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "update user failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, user); err != nil {
		h.logger.ErrorContext(r.Context(), "update user response failed", slog.String("error", err.Error()))
	}
}

func (h *UsersHandler) DeactivateUser(w nethttp.ResponseWriter, r *nethttp.Request) {
	if err := h.useCase.Deactivate(r.Context(), chi.URLParam(r, "userID")); err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "deactivate user failed", slog.String("error", err.Error()))
		}
		return
	}

	w.WriteHeader(nethttp.StatusNoContent)
}

func (h *UsersHandler) parseUserListFilter(r *nethttp.Request) (domain.UserListFilter, error) {
	query := r.URL.Query()

	filter := domain.UserListFilter{
		Search: query.Get("search"),
	}

	if roleValue := query.Get("role"); roleValue != "" {
		role := domain.UserRole(roleValue)
		filter.Role = &role
	}

	if isActiveValue := query.Get("is_active"); isActiveValue != "" {
		parsed, err := strconv.ParseBool(isActiveValue)
		if err != nil {
			return domain.UserListFilter{}, fmt.Errorf("is_active must be boolean")
		}
		filter.IsActive = &parsed
	}

	if limitValue := query.Get("limit"); limitValue != "" {
		parsed, err := strconv.Atoi(limitValue)
		if err != nil {
			return domain.UserListFilter{}, fmt.Errorf("limit must be integer")
		}
		filter.Limit = parsed
	}

	if offsetValue := query.Get("offset"); offsetValue != "" {
		parsed, err := strconv.Atoi(offsetValue)
		if err != nil {
			return domain.UserListFilter{}, fmt.Errorf("offset must be integer")
		}
		filter.Offset = parsed
	}

	return filter, nil
}
