package usecase

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"lms-arvand-backend/internal/domain"
)

type userRepository interface {
	Create(ctx context.Context, params domain.CreateUserParams) (domain.User, error)
	GetByID(ctx context.Context, userID string) (domain.User, error)
	List(ctx context.Context, filter domain.UserListFilter) ([]domain.User, int, error)
	Update(ctx context.Context, params domain.UpdateUserParams) (domain.User, error)
	Deactivate(ctx context.Context, userID string) error
}

type UserUseCase struct {
	repository userRepository
	bcryptCost int
	audit      *AuditLogger
}

func NewUserUseCase(repository userRepository, bcryptCost int) *UserUseCase {
	return &UserUseCase{
		repository: repository,
		bcryptCost: bcryptCost,
	}
}

func (u *UserUseCase) WithAudit(audit *AuditLogger) *UserUseCase {
	u.audit = audit
	return u
}

func (u *UserUseCase) Create(ctx context.Context, params domain.CreateUserParams) (domain.User, error) {
	normalized, err := normalizeCreateUserParams(params)
	if err != nil {
		return domain.User{}, fmt.Errorf("usecase users create: %w", err)
	}

	if normalized.Password != nil {
		passwordHash, err := hashPassword(*normalized.Password, u.bcryptCost)
		if err != nil {
			return domain.User{}, fmt.Errorf("usecase users create hash password: %w", err)
		}

		normalized.PasswordHash = &passwordHash
	}

	user, err := u.repository.Create(ctx, normalized)
	if err != nil {
		return domain.User{}, fmt.Errorf("usecase users create: %w", err)
	}

	if u.audit != nil {
		u.audit.Log(ctx, domain.AppEventUserCreated, map[string]any{
			"user_id":  user.ID,
			"email":    user.Email,
			"is_admin": user.Role == domain.UserRoleAdmin,
		})
	}

	return user, nil
}

func (u *UserUseCase) GetByID(ctx context.Context, userID string) (domain.User, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return domain.User{}, fmt.Errorf("usecase users get by id: %w", domain.ErrValidation)
	}

	user, err := u.repository.GetByID(ctx, userID)
	if err != nil {
		return domain.User{}, fmt.Errorf("usecase users get by id: %w", err)
	}
	if !user.IsActive {
		return domain.User{}, fmt.Errorf("usecase users get by id inactive: %w", domain.ErrNotFound)
	}

	return user, nil
}

func (u *UserUseCase) List(ctx context.Context, filter domain.UserListFilter) ([]domain.User, int, error) {
	normalized, err := normalizeUserListFilter(filter)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase users list: %w", err)
	}

	users, total, err := u.repository.List(ctx, normalized)
	if err != nil {
		return nil, 0, fmt.Errorf("usecase users list: %w", err)
	}

	return users, total, nil
}

func (u *UserUseCase) Update(ctx context.Context, params domain.UpdateUserParams) (domain.User, error) {
	normalized, err := normalizeUpdateUserParams(params)
	if err != nil {
		return domain.User{}, fmt.Errorf("usecase users update: %w", err)
	}

	if normalized.Password != nil {
		passwordHash, err := hashPassword(*normalized.Password, u.bcryptCost)
		if err != nil {
			return domain.User{}, fmt.Errorf("usecase users update hash password: %w", err)
		}

		normalized.PasswordHash = &passwordHash
	}

	user, err := u.repository.Update(ctx, normalized)
	if err != nil {
		return domain.User{}, fmt.Errorf("usecase users update: %w", err)
	}

	if u.audit != nil {
		u.audit.Log(ctx, domain.AppEventUserUpdated, map[string]any{
			"user_id":  user.ID,
			"email":    user.Email,
			"is_admin": user.Role == domain.UserRoleAdmin,
		})
	}

	return user, nil
}

func (u *UserUseCase) Deactivate(ctx context.Context, userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return fmt.Errorf("usecase users deactivate: %w", domain.ErrValidation)
	}

	if err := u.repository.Deactivate(ctx, userID); err != nil {
		return fmt.Errorf("usecase users deactivate: %w", err)
	}

	if u.audit != nil {
		u.audit.Log(ctx, domain.AppEventUserDeleted, map[string]any{
			"user_id": userID,
		})
	}

	return nil
}

func normalizeCreateUserParams(params domain.CreateUserParams) (domain.CreateUserParams, error) {
	params.FirstName = strings.TrimSpace(params.FirstName)
	params.LastName = strings.TrimSpace(params.LastName)
	params.Patronymic = strings.TrimSpace(params.Patronymic)
	params.Email = normalizeOptionalString(params.Email)
	params.GoogleID = normalizeOptionalString(params.GoogleID)
	params.Password = normalizeOptionalString(params.Password)
	params.PasswordHash = normalizeOptionalString(params.PasswordHash)
	params.Phone = normalizeOptionalString(params.Phone)
	params.BirthDate = normalizeOptionalString(params.BirthDate)
	params.Address = normalizeOptionalString(params.Address)
	params.City = normalizeOptionalString(params.City)
	params.AvatarURL = normalizeOptionalString(params.AvatarURL)

	var validation fieldValidationBuilder
	if params.Email == nil {
		validation.add("email", "required", "Email обязателен")
	}

	if params.Password != nil && len(*params.Password) < 8 {
		validation.add("password", "too_short", "Пароль должен быть минимум 8 символов")
	}

	if !params.Role.IsValid() {
		validation.add("role", "invalid_enum", "Роль должна быть admin, employee, student или guest")
	}

	if params.Gender == "" {
		params.Gender = domain.GenderUnspecified
	}

	if !params.Gender.IsValid() {
		validation.add("gender", "invalid_enum", "Пол должен быть male, female, other или unspecified")
	}

	addDateValidation(&validation, "birth_date", params.BirthDate, "Дата рождения")

	if err := validateEmail(params.Email); err != nil {
		validation.add("email", "invalid_email", "Email указан неверно")
	}

	if err := validateRolePayloads(params.Role, params.EmployeeInfo, params.StudentInfo, params.GuestInfo); err != nil {
		return domain.CreateUserParams{}, err
	}
	addStudentInfoDateValidation(&validation, params.StudentInfo)

	if err := validation.err(); err != nil {
		return domain.CreateUserParams{}, err
	}

	normalizeUserRoleDetails(&params)

	return params, nil
}

func normalizeUpdateUserParams(params domain.UpdateUserParams) (domain.UpdateUserParams, error) {
	params.ID = strings.TrimSpace(params.ID)
	params.FirstName = strings.TrimSpace(params.FirstName)
	params.LastName = strings.TrimSpace(params.LastName)
	params.Patronymic = strings.TrimSpace(params.Patronymic)
	params.Email = normalizeOptionalString(params.Email)
	params.GoogleID = normalizeOptionalString(params.GoogleID)
	params.Password = normalizeOptionalString(params.Password)
	params.PasswordHash = normalizeOptionalString(params.PasswordHash)
	params.Phone = normalizeOptionalString(params.Phone)
	params.BirthDate = normalizeOptionalString(params.BirthDate)
	params.Address = normalizeOptionalString(params.Address)
	params.City = normalizeOptionalString(params.City)
	params.AvatarURL = normalizeOptionalString(params.AvatarURL)

	var validation fieldValidationBuilder
	if params.Email == nil {
		validation.add("email", "required", "Email обязателен")
	}

	validation.addRequired("id", params.ID, "ID пользователя")
	if params.Password != nil && len(*params.Password) < 8 {
		validation.add("password", "too_short", "Пароль должен быть минимум 8 символов")
	}

	if !params.Role.IsValid() {
		validation.add("role", "invalid_enum", "Роль должна быть admin, employee, student или guest")
	}

	if params.Gender == "" {
		params.Gender = domain.GenderUnspecified
	}

	if !params.Gender.IsValid() {
		validation.add("gender", "invalid_enum", "Пол должен быть male, female, other или unspecified")
	}

	addDateValidation(&validation, "birth_date", params.BirthDate, "Дата рождения")

	if err := validateEmail(params.Email); err != nil {
		validation.add("email", "invalid_email", "Email указан неверно")
	}

	if err := validateRolePayloads(params.Role, params.EmployeeInfo, params.StudentInfo, params.GuestInfo); err != nil {
		return domain.UpdateUserParams{}, err
	}
	addStudentInfoDateValidation(&validation, params.StudentInfo)

	if err := validation.err(); err != nil {
		return domain.UpdateUserParams{}, err
	}

	normalizeUserRoleDetailsForUpdate(&params)

	return params, nil
}

func normalizeUserListFilter(filter domain.UserListFilter) (domain.UserListFilter, error) {
	filter.Search = strings.TrimSpace(filter.Search)

	if filter.Role != nil {
		role := domain.UserRole(strings.TrimSpace(string(*filter.Role)))
		if !role.IsValid() {
			return domain.UserListFilter{}, fmt.Errorf("role filter is invalid: %w", domain.ErrValidation)
		}
		filter.Role = &role
	}
	active := true
	filter.IsActive = &active

	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	if filter.Limit > 100 {
		filter.Limit = 100
	}

	if filter.Offset < 0 {
		return domain.UserListFilter{}, fmt.Errorf("offset must be non-negative: %w", domain.ErrValidation)
	}

	return filter, nil
}

func validateRolePayloads(
	role domain.UserRole,
	employeeInfo *domain.EmployeeInfo,
	studentInfo *domain.StudentInfo,
	guestInfo *domain.GuestInfo,
) error {
	var validation fieldValidationBuilder
	if role != domain.UserRoleEmployee && employeeInfo != nil {
		validation.add("employee_info", "forbidden_for_role", "employee_info можно передавать только для роли employee")
	}

	if role != domain.UserRoleStudent && studentInfo != nil {
		validation.add("student_info", "forbidden_for_role", "student_info можно передавать только для роли student")
	}

	if role != domain.UserRoleGuest && guestInfo != nil {
		validation.add("guest_info", "forbidden_for_role", "guest_info можно передавать только для роли guest")
	}

	return validation.err()
}

func normalizeUserRoleDetails(params *domain.CreateUserParams) {
	if params.EmployeeInfo != nil {
		params.EmployeeInfo.Branch = strings.TrimSpace(params.EmployeeInfo.Branch)
		params.EmployeeInfo.Office = strings.TrimSpace(params.EmployeeInfo.Office)
		params.EmployeeInfo.Position = strings.TrimSpace(params.EmployeeInfo.Position)
		params.EmployeeInfo.Department = strings.TrimSpace(params.EmployeeInfo.Department)
		params.EmployeeInfo.EmployeeID = strings.TrimSpace(params.EmployeeInfo.EmployeeID)
		params.EmployeeInfo.HireDate = strings.TrimSpace(params.EmployeeInfo.HireDate)
		params.EmployeeInfo.Notes = strings.TrimSpace(params.EmployeeInfo.Notes)
	}

	if params.StudentInfo != nil {
		params.StudentInfo.StudentID = strings.TrimSpace(params.StudentInfo.StudentID)
		params.StudentInfo.GroupName = strings.TrimSpace(params.StudentInfo.GroupName)
		params.StudentInfo.EducationLevel = strings.TrimSpace(params.StudentInfo.EducationLevel)
		params.StudentInfo.BirthDate = strings.TrimSpace(params.StudentInfo.BirthDate)
	}

	if params.GuestInfo != nil {
		params.GuestInfo.Source = strings.TrimSpace(params.GuestInfo.Source)
		params.GuestInfo.InvitedBy = normalizeOptionalString(params.GuestInfo.InvitedBy)
	}
}

func normalizeUserRoleDetailsForUpdate(params *domain.UpdateUserParams) {
	if params.EmployeeInfo != nil {
		params.EmployeeInfo.Branch = strings.TrimSpace(params.EmployeeInfo.Branch)
		params.EmployeeInfo.Office = strings.TrimSpace(params.EmployeeInfo.Office)
		params.EmployeeInfo.Position = strings.TrimSpace(params.EmployeeInfo.Position)
		params.EmployeeInfo.Department = strings.TrimSpace(params.EmployeeInfo.Department)
		params.EmployeeInfo.EmployeeID = strings.TrimSpace(params.EmployeeInfo.EmployeeID)
		params.EmployeeInfo.HireDate = strings.TrimSpace(params.EmployeeInfo.HireDate)
		params.EmployeeInfo.Notes = strings.TrimSpace(params.EmployeeInfo.Notes)
	}

	if params.StudentInfo != nil {
		params.StudentInfo.StudentID = strings.TrimSpace(params.StudentInfo.StudentID)
		params.StudentInfo.GroupName = strings.TrimSpace(params.StudentInfo.GroupName)
		params.StudentInfo.EducationLevel = strings.TrimSpace(params.StudentInfo.EducationLevel)
		params.StudentInfo.BirthDate = strings.TrimSpace(params.StudentInfo.BirthDate)
	}

	if params.GuestInfo != nil {
		params.GuestInfo.Source = strings.TrimSpace(params.GuestInfo.Source)
		params.GuestInfo.InvitedBy = normalizeOptionalString(params.GuestInfo.InvitedBy)
	}
}

func validateEmail(value *string) error {
	if value == nil {
		return nil
	}

	if _, err := mail.ParseAddress(*value); err != nil {
		return domain.ErrValidation
	}

	return nil
}

func addStudentInfoDateValidation(validation *fieldValidationBuilder, studentInfo *domain.StudentInfo) {
	if studentInfo == nil {
		return
	}
	addDateValidation(validation, "birth_date", &studentInfo.BirthDate, "Дата рождения")
}

func addDateValidation(validation *fieldValidationBuilder, field string, value *string, label string) {
	if validation == nil || value == nil || strings.TrimSpace(*value) == "" {
		return
	}

	if _, err := time.Parse("2006-01-02", strings.TrimSpace(*value)); err != nil {
		validation.add(field, "invalid_date", label+" должна быть в формате YYYY-MM-DD")
	}
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func normalizeStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))

	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}

		if _, exists := seen[trimmed]; exists {
			continue
		}

		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	if len(normalized) == 0 {
		return nil
	}

	return normalized
}
