package http

import (
	"context"
	"fmt"
	"log/slog"
	nethttp "net/http"
	"strconv"
	"strings"

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
	ID           string               `json:"id,omitempty"`
	Username     string               `json:"username,omitempty"`
	Email        *string              `json:"email,omitempty"`
	GoogleID     *string              `json:"google_id,omitempty"`
	Password     *string              `json:"password,omitempty"`
	Role         domain.UserRole      `json:"role"`
	FirstName    string               `json:"first_name"`
	LastName     string               `json:"last_name"`
	Patronymic   string               `json:"patronymic,omitempty"`
	Phone        *string              `json:"phone,omitempty"`
	Gender       domain.Gender        `json:"gender"`
	Address      *string              `json:"address,omitempty"`
	City         *string              `json:"city,omitempty"`
	AvatarURL    *string              `json:"avatar_url,omitempty"`
	BirthDate    *string              `json:"birth_date,omitempty"`
	IsActive     *bool                `json:"is_active,omitempty"`
	EmployeeInfo *domain.EmployeeInfo `json:"employee_info,omitempty"`
	AdminInfo    *domain.AdminInfo    `json:"admin_info,omitempty"`
	StudentInfo  *domain.StudentInfo  `json:"student_info,omitempty"`
	GuestInfo    *domain.GuestInfo    `json:"guest_info,omitempty"`
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
		writeError(w, nethttp.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}

	params := request.toCreateParams()

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
	var request userRequest
	if err := decodeJSON(w, r, &request, 1<<20); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid_json", "invalid request body")
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

func (r userRequest) toCreateParams() domain.CreateUserParams {
	role := normalizeUserRequestRole(r.Role)
	return domain.CreateUserParams{
		Username:     fallbackUsername(r.Username, r.Email),
		Email:        r.Email,
		GoogleID:     r.GoogleID,
		Password:     r.Password,
		Role:         role,
		FirstName:    r.FirstName,
		LastName:     r.LastName,
		Patronymic:   r.Patronymic,
		Phone:        r.Phone,
		Gender:       r.Gender,
		Address:      r.Address,
		City:         r.City,
		AvatarURL:    r.AvatarURL,
		EmployeeInfo: infoForEmployee(role, r.EmployeeInfo),
		AdminInfo:    infoForAdmin(role, r.AdminInfo),
		StudentInfo:  infoForStudent(role, r.StudentInfo, r.BirthDate),
		GuestInfo:    infoForGuest(role, r.GuestInfo),
	}
}

func (r userRequest) toUpdateParams() domain.UpdateUserParams {
	role := normalizeUserRequestRole(r.Role)
	isActive := true
	if r.IsActive != nil {
		isActive = *r.IsActive
	}

	return domain.UpdateUserParams{
		ID:           r.ID,
		Username:     fallbackUsername(r.Username, r.Email),
		Email:        r.Email,
		GoogleID:     r.GoogleID,
		Password:     r.Password,
		Role:         role,
		FirstName:    r.FirstName,
		LastName:     r.LastName,
		Patronymic:   r.Patronymic,
		Phone:        r.Phone,
		Gender:       r.Gender,
		Address:      r.Address,
		City:         r.City,
		AvatarURL:    r.AvatarURL,
		IsActive:     isActive,
		EmployeeInfo: infoForEmployee(role, r.EmployeeInfo),
		AdminInfo:    infoForAdmin(role, r.AdminInfo),
		StudentInfo:  infoForStudent(role, r.StudentInfo, r.BirthDate),
		GuestInfo:    infoForGuest(role, r.GuestInfo),
	}
}

func normalizeUserRequestRole(role domain.UserRole) domain.UserRole {
	switch domain.UserRole(strings.TrimSpace(string(role))) {
	case "", "user":
		return domain.UserRoleStudent
	default:
		return domain.UserRole(strings.TrimSpace(string(role)))
	}
}

func fallbackUsername(username string, email *string) string {
	username = strings.TrimSpace(username)
	if username != "" {
		return username
	}

	if email == nil {
		return ""
	}

	return strings.TrimSpace(*email)
}

func infoForEmployee(role domain.UserRole, value *domain.EmployeeInfo) *domain.EmployeeInfo {
	if role == domain.UserRoleEmployee {
		return value
	}

	return nil
}

func infoForAdmin(role domain.UserRole, value *domain.AdminInfo) *domain.AdminInfo {
	if role == domain.UserRoleAdmin {
		return value
	}

	return nil
}

func infoForStudent(role domain.UserRole, value *domain.StudentInfo, birthDate *string) *domain.StudentInfo {
	if role != domain.UserRoleStudent {
		return nil
	}

	if value == nil {
		value = &domain.StudentInfo{}
	}

	if birthDate != nil {
		value.BirthDate = strings.TrimSpace(*birthDate)
	}

	return value
}

func infoForGuest(role domain.UserRole, value *domain.GuestInfo) *domain.GuestInfo {
	if role == domain.UserRoleGuest {
		return value
	}

	return nil
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
