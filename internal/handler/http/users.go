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

type userRequest struct {
	ID         string  `json:"id,omitempty"`
	Email      *string `json:"email,omitempty"`
	GoogleID   *string `json:"google_id,omitempty"`
	Password   *string `json:"password,omitempty"`
	IsAdmin    *bool   `json:"is_admin,omitempty"`
	FirstName  string  `json:"first_name"`
	LastName   string  `json:"last_name"`
	Patronymic string  `json:"patronymic,omitempty"`
	Phone      *string `json:"phone,omitempty"`
	IsMale     *bool   `json:"is_male,omitempty"`
	Address    *string `json:"address,omitempty"`
	City       *string `json:"city,omitempty"`
	AvatarURL  *string `json:"avatar_url,omitempty"`
	BirthDate  *string `json:"birth_date,omitempty"`
}

func NewUsersHandler(logger *slog.Logger, useCase userUseCase) *UsersHandler {
	return &UsersHandler{
		logger:  logger,
		useCase: useCase,
	}
}

func (h *UsersHandler) CreateUser(w nethttp.ResponseWriter, r *nethttp.Request) {
	var request userRequest
	if err := decodeJSON(w, r, &request, 1<<20); err != nil {
		writeDecodeError(w, err)
		return
	}

	user, err := h.useCase.Create(r.Context(), request.toCreateParams())
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "create user failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusCreated, toUserResponse(user)); err != nil {
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

	if err := writeJSON(w, nethttp.StatusOK, toUserResponse(user)); err != nil {
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

	if err := writePagedJSON(w, nethttp.StatusOK, toUserResponses(users), total, filter.Limit, filter.Offset); err != nil {
		h.logger.ErrorContext(r.Context(), "list users response failed", slog.String("error", err.Error()))
	}
}

func (h *UsersHandler) UpdateUser(w nethttp.ResponseWriter, r *nethttp.Request) {
	var request userRequest
	if err := decodeJSON(w, r, &request, 1<<20); err != nil {
		writeDecodeError(w, err)
		return
	}

	params := request.toUpdateParams()
	params.ID = chi.URLParam(r, "userID")

	user, err := h.useCase.Update(r.Context(), params)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "update user failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, toUserResponse(user)); err != nil {
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

func (r userRequest) toCreateParams() domain.CreateUserParams {
	return domain.CreateUserParams{
		Email:      r.Email,
		GoogleID:   r.GoogleID,
		Password:   r.Password,
		IsAdmin:    boolFromPointer(r.IsAdmin),
		FirstName:  r.FirstName,
		LastName:   r.LastName,
		Patronymic: r.Patronymic,
		Phone:      r.Phone,
		IsMale:     r.IsMale,
		BirthDate:  r.BirthDate,
		Address:    r.Address,
		City:       r.City,
		AvatarURL:  r.AvatarURL,
	}
}

func (r userRequest) toUpdateParams() domain.UpdateUserParams {
	return domain.UpdateUserParams{
		ID:         r.ID,
		Email:      r.Email,
		GoogleID:   r.GoogleID,
		Password:   r.Password,
		IsAdmin:    boolFromPointer(r.IsAdmin),
		FirstName:  r.FirstName,
		LastName:   r.LastName,
		Patronymic: r.Patronymic,
		Phone:      r.Phone,
		IsMale:     r.IsMale,
		BirthDate:  r.BirthDate,
		Address:    r.Address,
		City:       r.City,
		AvatarURL:  r.AvatarURL,
		IsActive:   true,
	}
}

func boolFromPointer(v *bool) bool {
	if v == nil {
		return false
	}
	return *v
}

func (h *UsersHandler) parseUserListFilter(r *nethttp.Request) (domain.UserListFilter, error) {
	query := r.URL.Query()

	filter := domain.UserListFilter{
		Search: query.Get("search"),
	}

	if isAdminValue := query.Get("is_admin"); isAdminValue != "" {
		v, err := strconv.ParseBool(isAdminValue)
		if err != nil {
			return domain.UserListFilter{}, fmt.Errorf("is_admin must be true or false")
		}
		filter.IsAdmin = &v
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
