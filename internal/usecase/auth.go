package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"lms-arvand-backend/internal/domain"
)

type authUserRepository interface {
	GetByID(ctx context.Context, userID string) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	GetByGoogleID(ctx context.Context, googleID string) (domain.User, error)
	LinkGoogleID(ctx context.Context, userID, googleID string) (domain.User, error)
	Create(ctx context.Context, params domain.CreateUserParams) (domain.User, error)
	Update(ctx context.Context, params domain.UpdateUserParams) (domain.User, error)
}

type authSessionRepository interface {
	Create(ctx context.Context, params domain.CreateSessionParams) (domain.Session, error)
	GetByToken(ctx context.Context, token string) (domain.Session, error)
	GetByTokenWithUser(ctx context.Context, token string) (domain.AuthIdentity, error)
	DeleteByToken(ctx context.Context, token string) error
}

type authSessionCache interface {
	Get(token string) (domain.AuthIdentity, bool)
	Set(token string, identity domain.AuthIdentity, expiresAt *time.Time)
	Delete(token string)
}

type authLoginAttemptRepository interface {
	CountRecentFailed(ctx context.Context, identifier string, ipAddress *string, scope string, since time.Time) (int, error)
	Record(ctx context.Context, identifier string, ipAddress *string, succeeded bool) error
}

type authPasswordResetRepository interface {
	Create(ctx context.Context, params domain.CreatePasswordResetTokenParams) (domain.PasswordResetToken, error)
	GetValidByTokenHash(ctx context.Context, tokenHash string, now time.Time) (domain.PasswordResetToken, error)
	MarkUsed(ctx context.Context, tokenID string) error
}

type AuthUseCase struct {
	users               authUserRepository
	sessions            authSessionRepository
	sessionCache        authSessionCache
	loginAttempts       authLoginAttemptRepository
	passwordResets      authPasswordResetRepository
	sessionTTL          time.Duration
	bcryptCost          int
	loginLockoutEnabled bool
	maxLoginAttempts    int
	loginAttemptWindow  time.Duration
	loginLockoutScope   string
	passwordResetTTL    time.Duration
	passwordResetReturn bool
	google              googleTokenVerifier
	googleDefaultRole   domain.UserRole
	audit               *AuditLogger
	now                 func() time.Time
}

func NewAuthUseCase(
	users authUserRepository,
	sessions authSessionRepository,
	sessionCache authSessionCache,
	sessionTTL time.Duration,
	bcryptCost int,
	loginAttempts authLoginAttemptRepository,
	passwordResets authPasswordResetRepository,
	loginLockoutEnabled bool,
	maxLoginAttempts int,
	loginAttemptWindow time.Duration,
	loginLockoutScope string,
	passwordResetTTL time.Duration,
	passwordResetReturn bool,
	google googleTokenVerifier,
	googleDefaultRole string,
) *AuthUseCase {
	return &AuthUseCase{
		users:               users,
		sessions:            sessions,
		sessionCache:        sessionCache,
		loginAttempts:       loginAttempts,
		passwordResets:      passwordResets,
		sessionTTL:          sessionTTL,
		bcryptCost:          bcryptCost,
		loginLockoutEnabled: loginLockoutEnabled,
		maxLoginAttempts:    maxLoginAttempts,
		loginAttemptWindow:  loginAttemptWindow,
		loginLockoutScope:   normalizeLoginLockoutScope(loginLockoutScope),
		passwordResetTTL:    passwordResetTTL,
		passwordResetReturn: passwordResetReturn,
		google:              google,
		googleDefaultRole:   normalizeGoogleDefaultRole(googleDefaultRole),
		now:                 time.Now,
	}
}

func (u *AuthUseCase) WithAudit(audit *AuditLogger) *AuthUseCase {
	u.audit = audit
	return u
}

func (u *AuthUseCase) Register(ctx context.Context, params domain.RegisterParams) (domain.LoginResult, error) {
	normalized, err := normalizeRegisterParams(params)
	if err != nil {
		return domain.LoginResult{}, fmt.Errorf("usecase auth register: %w", err)
	}

	passwordHash, err := hashPassword(normalized.Password, u.bcryptCost)
	if err != nil {
		return domain.LoginResult{}, fmt.Errorf("usecase auth register hash password: %w", err)
	}

	email := normalized.Email
	passwordHashPtr := passwordHash
	user, err := u.users.Create(ctx, domain.CreateUserParams{
		Email:        &email,
		PasswordHash: &passwordHashPtr,
		Role:         domain.UserRoleStudent,
		FirstName:    normalized.FirstName,
		LastName:     normalized.LastName,
		Patronymic:   normalized.Patronymic,
		Phone:        normalized.Phone,
		Gender:       normalized.Gender,
		BirthDate:    normalized.BirthDate,
		Address:      normalized.Address,
		City:         normalized.City,
		AvatarURL:    normalized.AvatarURL,
		StudentInfo: &domain.StudentInfo{
			BirthDate: stringFromPointer(normalized.BirthDate),
		},
	})
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return domain.LoginResult{}, domain.ConflictError("Пользователь с таким email или логином уже существует")
		}
		return domain.LoginResult{}, fmt.Errorf("usecase auth register create user: %w", err)
	}

	if u.audit != nil {
		u.audit.Log(ctx, domain.AppEventUserCreated, map[string]any{
			"user_id":  user.ID,
			"email":    user.Email,
			"is_admin": user.Role == domain.UserRoleAdmin,
			"source":   "register",
		})
	}

	session, err := u.createSession(ctx, user.ID, normalized.IPAddress, normalized.UserAgent)
	if err != nil {
		return domain.LoginResult{}, fmt.Errorf("usecase auth register create session: %w", err)
	}

	u.cacheIdentity(session.Token, user, session)
	return loginResultFromSession(user, session), nil
}

func (u *AuthUseCase) Login(ctx context.Context, params domain.LoginParams) (domain.LoginResult, error) {
	normalized, err := normalizeLoginParams(params)
	if err != nil {
		return domain.LoginResult{}, fmt.Errorf("usecase auth login: %w", err)
	}

	succeeded := false
	defer u.recordLoginAttempt(ctx, normalized.Identifier, normalized.IPAddress, &succeeded)

	if err := u.ensureLoginAllowed(ctx, normalized.Identifier, normalized.IPAddress); err != nil {
		return domain.LoginResult{}, fmt.Errorf("usecase auth login rate limit: %w", err)
	}

	user, err := u.users.GetByEmail(ctx, normalized.Identifier)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.LoginResult{}, domain.UnauthorizedError("Неверный логин или пароль")
		}

		return domain.LoginResult{}, fmt.Errorf("usecase auth login get user: %w", err)
	}

	if !user.IsActive {
		return domain.LoginResult{}, domain.UnauthorizedError("Пользователь отключен")
	}

	if user.PasswordHash == nil {
		return domain.LoginResult{}, domain.UnauthorizedError("Для этого аккаунта вход по паролю недоступен")
	}

	if err := comparePasswordHash(*user.PasswordHash, normalized.Password); err != nil {
		return domain.LoginResult{}, domain.UnauthorizedError("Неверный логин или пароль")
	}

	token, err := generateSessionToken()
	if err != nil {
		return domain.LoginResult{}, fmt.Errorf("usecase auth login generate token: %w", err)
	}

	var expiresAt *time.Time
	if u.sessionTTL > 0 {
		expiration := u.now().UTC().Add(u.sessionTTL)
		expiresAt = &expiration
	}

	session, err := u.sessions.Create(ctx, domain.CreateSessionParams{
		Token:     token,
		UserID:    user.ID,
		IPAddress: normalized.IPAddress,
		UserAgent: normalized.UserAgent,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return domain.LoginResult{}, fmt.Errorf("usecase auth login create session: %w", err)
	}

	u.cacheIdentity(token, user, session)
	succeeded = true
	return loginResultFromSession(user, session), nil
}

func (u *AuthUseCase) LoginWithGoogle(ctx context.Context, params domain.GoogleLoginParams) (domain.LoginResult, error) {
	normalized, err := normalizeGoogleLoginParams(params)
	if err != nil {
		return domain.LoginResult{}, fmt.Errorf("usecase auth google login: %w", err)
	}

	if u.google == nil {
		return domain.LoginResult{}, fmt.Errorf("usecase auth google login: %w", domain.UnavailableError("Вход через Google не настроен"))
	}

	identity, err := u.google.VerifyIDToken(ctx, normalized.IDToken)
	if err != nil {
		return domain.LoginResult{}, fmt.Errorf("usecase auth google login verify token: %w", err)
	}

	user, err := u.resolveGoogleUser(ctx, identity)
	if err != nil {
		return domain.LoginResult{}, fmt.Errorf("usecase auth google login resolve user: %w", err)
	}

	if !user.IsActive {
		return domain.LoginResult{}, fmt.Errorf("usecase auth google login inactive user: %w", domain.ErrUnauthorized)
	}

	session, err := u.createSession(ctx, user.ID, normalized.IPAddress, normalized.UserAgent)
	if err != nil {
		return domain.LoginResult{}, fmt.Errorf("usecase auth google login create session: %w", err)
	}

	u.cacheIdentity(session.Token, user, session)
	return loginResultFromSession(user, session), nil
}

func (u *AuthUseCase) Authenticate(ctx context.Context, token string) (domain.AuthIdentity, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return domain.AuthIdentity{}, fmt.Errorf("usecase auth authenticate: %w", domain.ErrUnauthorized)
	}

	if identity, ok := u.getCachedIdentity(token); ok {
		if identity.Session.ExpiresAt != nil && identity.Session.ExpiresAt.Before(u.now().UTC()) {
			u.deleteCachedIdentity(token)
			return domain.AuthIdentity{}, fmt.Errorf("usecase auth authenticate expired session: %w", domain.ErrUnauthorized)
		}

		if !identity.User.IsActive {
			u.deleteCachedIdentity(token)
			return domain.AuthIdentity{}, fmt.Errorf("usecase auth authenticate inactive user: %w", domain.ErrUnauthorized)
		}

		return identity, nil
	}

	identity, err := u.sessions.GetByTokenWithUser(ctx, token)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.AuthIdentity{}, fmt.Errorf("usecase auth authenticate: %w", domain.ErrUnauthorized)
		}

		return domain.AuthIdentity{}, fmt.Errorf("usecase auth authenticate get session: %w", err)
	}

	if identity.Session.ExpiresAt != nil && identity.Session.ExpiresAt.Before(u.now().UTC()) {
		return domain.AuthIdentity{}, fmt.Errorf("usecase auth authenticate expired session: %w", domain.ErrUnauthorized)
	}

	if !identity.User.IsActive {
		return domain.AuthIdentity{}, fmt.Errorf("usecase auth authenticate inactive user: %w", domain.ErrUnauthorized)
	}

	u.cacheIdentity(token, identity.User, identity.Session)
	return identity, nil
}

func (u *AuthUseCase) Logout(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("usecase auth logout: %w", domain.ErrUnauthorized)
	}

	if err := u.sessions.DeleteByToken(ctx, token); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("usecase auth logout: %w", domain.ErrUnauthorized)
		}

		return fmt.Errorf("usecase auth logout delete session: %w", err)
	}

	u.deleteCachedIdentity(token)
	return nil
}

func (u *AuthUseCase) UpdateProfile(ctx context.Context, params domain.UpdateProfileParams) (domain.User, error) {
	normalized, err := normalizeUpdateProfileParams(params)
	if err != nil {
		return domain.User{}, fmt.Errorf("usecase auth update profile: %w", err)
	}

	current, err := u.users.GetByID(ctx, normalized.UserID)
	if err != nil {
		return domain.User{}, fmt.Errorf("usecase auth update profile get current user: %w", err)
	}

	update := profileUpdateToUserParams(current, normalized)
	if update.Password != nil {
		passwordHash, err := hashPassword(*update.Password, u.bcryptCost)
		if err != nil {
			return domain.User{}, fmt.Errorf("usecase auth update profile hash password: %w", err)
		}
		update.PasswordHash = &passwordHash
	}

	user, err := u.users.Update(ctx, update)
	if err != nil {
		return domain.User{}, fmt.Errorf("usecase auth update profile save: %w", err)
	}

	if u.audit != nil {
		u.audit.Log(ctx, domain.AppEventUserUpdated, map[string]any{
			"user_id": user.ID,
			"email":   user.Email,
			"source":  "profile",
		})
	}

	if normalized.SessionToken != "" && u.sessionCache != nil {
		if identity, ok := u.getCachedIdentity(normalized.SessionToken); ok {
			identity.User = user
			u.sessionCache.Set(normalized.SessionToken, identity, identity.Session.ExpiresAt)
		}
	}

	return user, nil
}

func (u *AuthUseCase) ChangePassword(ctx context.Context, params domain.ChangePasswordParams) error {
	normalized, err := normalizeChangePasswordParams(params)
	if err != nil {
		return fmt.Errorf("usecase auth change password: %w", err)
	}

	current, err := u.users.GetByID(ctx, normalized.UserID)
	if err != nil {
		return fmt.Errorf("usecase auth change password get current user: %w", err)
	}

	if !current.IsActive {
		return fmt.Errorf("usecase auth change password inactive user: %w", domain.ErrUnauthorized)
	}

	if current.PasswordHash != nil {
		if normalized.CurrentPassword == nil {
			return domain.FieldValidationError("Проверьте поля формы",
				domain.ValidationField("current_password", "required", "Текущий пароль обязателен"))
		}
		if err := comparePasswordHash(*current.PasswordHash, *normalized.CurrentPassword); err != nil {
			return domain.UnauthorizedError("Текущий пароль указан неверно")
		}
	}

	passwordHash, err := hashPassword(normalized.NewPassword, u.bcryptCost)
	if err != nil {
		return fmt.Errorf("usecase auth change password hash password: %w", err)
	}

	update := userToPasswordUpdateParams(current)
	update.PasswordHash = &passwordHash
	if _, err := u.users.Update(ctx, update); err != nil {
		return fmt.Errorf("usecase auth change password save: %w", err)
	}

	u.deleteCachedIdentity(normalized.SessionToken)
	if u.audit != nil {
		u.audit.Log(ctx, domain.AppEventUserUpdated, map[string]any{
			"user_id": current.ID,
			"email":   current.Email,
			"source":  "password_change",
		})
	}

	return nil
}

func (u *AuthUseCase) ForgotPassword(ctx context.Context, params domain.ForgotPasswordParams) (domain.ForgotPasswordResult, error) {
	normalized, err := normalizeForgotPasswordParams(params)
	if err != nil {
		return domain.ForgotPasswordResult{}, fmt.Errorf("usecase auth forgot password: %w", err)
	}

	if u.passwordResets == nil {
		return domain.ForgotPasswordResult{}, fmt.Errorf("usecase auth forgot password: %w", domain.UnavailableError("Сброс пароля не настроен"))
	}

	ttl := u.passwordResetTTL
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	expiresAt := u.now().UTC().Add(ttl)

	user, err := u.users.GetByEmail(ctx, normalized.Email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ForgotPasswordResult{ExpiresAt: &expiresAt}, nil
		}
		return domain.ForgotPasswordResult{}, fmt.Errorf("usecase auth forgot password get user: %w", err)
	}

	if !user.IsActive {
		return domain.ForgotPasswordResult{ExpiresAt: &expiresAt}, nil
	}

	token, err := generateSessionToken()
	if err != nil {
		return domain.ForgotPasswordResult{}, fmt.Errorf("usecase auth forgot password generate token: %w", err)
	}

	if _, err := u.passwordResets.Create(ctx, domain.CreatePasswordResetTokenParams{
		UserID:    user.ID,
		TokenHash: hashPasswordResetToken(token),
		ExpiresAt: expiresAt,
	}); err != nil {
		return domain.ForgotPasswordResult{}, fmt.Errorf("usecase auth forgot password create token: %w", err)
	}

	result := domain.ForgotPasswordResult{ExpiresAt: &expiresAt}
	if u.passwordResetReturn {
		result.ResetToken = &token
	}

	return result, nil
}

func (u *AuthUseCase) ResetPassword(ctx context.Context, params domain.ResetPasswordParams) error {
	normalized, err := normalizeResetPasswordParams(params)
	if err != nil {
		return fmt.Errorf("usecase auth reset password: %w", err)
	}

	if u.passwordResets == nil {
		return fmt.Errorf("usecase auth reset password: %w", domain.UnavailableError("Сброс пароля не настроен"))
	}

	resetToken, err := u.passwordResets.GetValidByTokenHash(ctx, hashPasswordResetToken(normalized.Token), u.now().UTC())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.UnauthorizedError("Ссылка сброса пароля недействительна или истекла")
		}
		return fmt.Errorf("usecase auth reset password get token: %w", err)
	}

	user, err := u.users.GetByID(ctx, resetToken.UserID)
	if err != nil {
		return fmt.Errorf("usecase auth reset password get user: %w", err)
	}
	if !user.IsActive {
		return fmt.Errorf("usecase auth reset password inactive user: %w", domain.ErrUnauthorized)
	}

	passwordHash, err := hashPassword(normalized.NewPassword, u.bcryptCost)
	if err != nil {
		return fmt.Errorf("usecase auth reset password hash password: %w", err)
	}

	update := userToPasswordUpdateParams(user)
	update.PasswordHash = &passwordHash
	if _, err := u.users.Update(ctx, update); err != nil {
		return fmt.Errorf("usecase auth reset password save: %w", err)
	}

	if err := u.passwordResets.MarkUsed(ctx, resetToken.ID); err != nil {
		return fmt.Errorf("usecase auth reset password mark token used: %w", err)
	}

	if u.audit != nil {
		u.audit.Log(ctx, domain.AppEventUserUpdated, map[string]any{
			"user_id": user.ID,
			"email":   user.Email,
			"source":  "password_reset",
		})
	}

	return nil
}

func (u *AuthUseCase) resolveGoogleUser(ctx context.Context, identity GoogleIdentity) (domain.User, error) {
	user, err := u.users.GetByGoogleID(ctx, identity.Subject)
	if err == nil {
		return user, nil
	}

	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, fmt.Errorf("get by google id: %w", err)
	}

	user, err = u.users.GetByEmail(ctx, identity.Email)
	if err == nil {
		if user.GoogleID != nil && *user.GoogleID != identity.Subject {
			return domain.User{}, domain.ConflictError("Этот Google-аккаунт уже привязан к другому пользователю")
		}

		linkedUser, err := u.users.LinkGoogleID(ctx, user.ID, identity.Subject)
		if err != nil {
			return domain.User{}, fmt.Errorf("link google id: %w", err)
		}

		return linkedUser, nil
	}

	if !errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, fmt.Errorf("get by email: %w", err)
	}

	googleID := identity.Subject
	email := identity.Email
	avatarURL := normalizeOptionalString(&identity.Picture)

	createdUser, err := u.users.Create(ctx, domain.CreateUserParams{
		Email:     &email,
		GoogleID:  &googleID,
		Role:      u.googleDefaultRole,
		FirstName: identity.GivenName,
		LastName:  identity.FamilyName,
		Gender:    domain.GenderUnspecified,
		AvatarURL: avatarURL,
	})
	if err != nil {
		return domain.User{}, fmt.Errorf("create google user: %w", err)
	}

	return createdUser, nil
}

func normalizeGoogleDefaultRole(role string) domain.UserRole {
	normalized := domain.UserRole(strings.ToLower(strings.TrimSpace(role)))
	if normalized.IsValid() {
		return normalized
	}

	return domain.UserRoleStudent
}

func (u *AuthUseCase) createSession(
	ctx context.Context,
	userID string,
	ipAddress *string,
	userAgent *string,
) (domain.Session, error) {
	token, err := generateSessionToken()
	if err != nil {
		return domain.Session{}, fmt.Errorf("generate session token: %w", err)
	}

	var expiresAt *time.Time
	if u.sessionTTL > 0 {
		expiration := u.now().UTC().Add(u.sessionTTL)
		expiresAt = &expiration
	}

	session, err := u.sessions.Create(ctx, domain.CreateSessionParams{
		Token:     token,
		UserID:    userID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return domain.Session{}, err
	}

	return session, nil
}

func (u *AuthUseCase) cacheIdentity(token string, user domain.User, session domain.Session) {
	if u.sessionCache == nil || token == "" {
		return
	}

	cachedUser := user
	cachedUser.PasswordHash = nil

	u.sessionCache.Set(token, domain.AuthIdentity{
		User:    cachedUser,
		Session: session,
	}, session.ExpiresAt)
}

func (u *AuthUseCase) getCachedIdentity(token string) (domain.AuthIdentity, bool) {
	if u.sessionCache == nil || token == "" {
		return domain.AuthIdentity{}, false
	}

	return u.sessionCache.Get(token)
}

func (u *AuthUseCase) deleteCachedIdentity(token string) {
	if u.sessionCache == nil || token == "" {
		return
	}

	u.sessionCache.Delete(token)
}

func normalizeGoogleLoginParams(params domain.GoogleLoginParams) (domain.GoogleLoginParams, error) {
	params.IDToken = strings.TrimSpace(params.IDToken)
	params.IPAddress = normalizeOptionalString(params.IPAddress)
	params.UserAgent = normalizeOptionalString(params.UserAgent)

	if params.IDToken == "" {
		return domain.GoogleLoginParams{}, domain.FieldValidationError("Проверьте поля формы",
			domain.ValidationField("id_token", "required", "Google ID token обязателен"))
	}

	return params, nil
}

func normalizeRegisterParams(params domain.RegisterParams) (domain.RegisterParams, error) {
	params.Email = strings.TrimSpace(strings.ToLower(params.Email))
	params.Password = strings.TrimSpace(params.Password)
	params.FirstName = strings.TrimSpace(params.FirstName)
	params.LastName = strings.TrimSpace(params.LastName)
	params.Patronymic = strings.TrimSpace(params.Patronymic)
	params.Phone = normalizeOptionalString(params.Phone)
	params.Address = normalizeOptionalString(params.Address)
	params.City = normalizeOptionalString(params.City)
	params.AvatarURL = normalizeOptionalString(params.AvatarURL)
	params.BirthDate = normalizeOptionalString(params.BirthDate)
	params.IPAddress = normalizeOptionalString(params.IPAddress)
	params.UserAgent = normalizeOptionalString(params.UserAgent)
	params.Gender = genderFromBoolean(params.Gender, params.IsMale)

	var validation fieldValidationBuilder
	validation.addRequired("email", params.Email, "Email")
	validation.addRequired("password", params.Password, "Пароль")
	validation.addRequired("first_name", params.FirstName, "Имя")
	validation.addRequired("last_name", params.LastName, "Фамилия")

	if params.Email != "" {
		emailCopy := params.Email
		if err := validateEmail(&emailCopy); err != nil {
			validation.add("email", "invalid_email", "Email указан неверно")
		}
	}

	if params.Password != "" && len(params.Password) < 8 {
		validation.add("password", "too_short", "Пароль должен быть минимум 8 символов")
	}

	if params.Gender == "" {
		params.Gender = domain.GenderUnspecified
	}
	if !params.Gender.IsValid() {
		validation.add("gender", "invalid_enum", "Пол должен быть male, female, other или unspecified")
	}

	addDateValidation(&validation, "birth_date", params.BirthDate, "Дата рождения")

	if err := validation.err(); err != nil {
		return domain.RegisterParams{}, err
	}

	return params, nil
}

func normalizeUpdateProfileParams(params domain.UpdateProfileParams) (domain.UpdateProfileParams, error) {
	params.UserID = strings.TrimSpace(params.UserID)
	params.SessionToken = strings.TrimSpace(params.SessionToken)
	params.Email = normalizeOptionalString(params.Email)
	params.Password = normalizeOptionalString(params.Password)
	params.FirstName = strings.TrimSpace(params.FirstName)
	params.LastName = strings.TrimSpace(params.LastName)
	params.Patronymic = strings.TrimSpace(params.Patronymic)
	params.Phone = normalizeOptionalString(params.Phone)
	params.Address = normalizeOptionalString(params.Address)
	params.City = normalizeOptionalString(params.City)
	params.AvatarURL = normalizeOptionalString(params.AvatarURL)
	params.BirthDate = normalizeOptionalString(params.BirthDate)
	params.Gender = genderFromBoolean(params.Gender, params.IsMale)

	var validation fieldValidationBuilder
	validation.addRequired("user_id", params.UserID, "ID пользователя")

	if params.Email != nil {
		validation.add("email", "forbidden", "Email нельзя менять через профиль")
	}
	if params.Password != nil {
		validation.add("password", "forbidden", "Для смены пароля используйте /api/v1/auth/password/change")
	}
	if params.Gender != "" && !params.Gender.IsValid() {
		validation.add("gender", "invalid_enum", "Пол должен быть male, female, other или unspecified")
	}

	addDateValidation(&validation, "birth_date", params.BirthDate, "Дата рождения")

	if err := validation.err(); err != nil {
		return domain.UpdateProfileParams{}, err
	}

	return params, nil
}

func loginResultFromSession(user domain.User, session domain.Session) domain.LoginResult {
	return domain.LoginResult{
		Token:     session.Token,
		ExpiresAt: session.ExpiresAt,
		User:      user,
	}
}

func normalizeLoginParams(params domain.LoginParams) (domain.LoginParams, error) {
	params.Identifier = strings.TrimSpace(params.Identifier)
	params.Password = strings.TrimSpace(params.Password)
	params.IPAddress = normalizeOptionalString(params.IPAddress)
	params.UserAgent = normalizeOptionalString(params.UserAgent)

	var validation fieldValidationBuilder
	if params.Identifier == "" {
		validation.add("identifier", "required", "Email или логин обязателен")
	}

	if params.Password == "" {
		validation.add("password", "required", "Пароль обязателен")
	}

	if err := validation.err(); err != nil {
		return domain.LoginParams{}, err
	}

	return params, nil
}

func normalizeChangePasswordParams(params domain.ChangePasswordParams) (domain.ChangePasswordParams, error) {
	params.UserID = strings.TrimSpace(params.UserID)
	params.SessionToken = strings.TrimSpace(params.SessionToken)
	params.CurrentPassword = normalizeOptionalString(params.CurrentPassword)
	params.NewPassword = strings.TrimSpace(params.NewPassword)

	var validation fieldValidationBuilder
	validation.addRequired("user_id", params.UserID, "ID пользователя")
	validation.addRequired("new_password", params.NewPassword, "Новый пароль")
	if params.NewPassword != "" && len(params.NewPassword) < 8 {
		validation.add("new_password", "too_short", "Новый пароль должен быть минимум 8 символов")
	}

	if err := validation.err(); err != nil {
		return domain.ChangePasswordParams{}, err
	}

	return params, nil
}

func normalizeForgotPasswordParams(params domain.ForgotPasswordParams) (domain.ForgotPasswordParams, error) {
	params.Email = strings.TrimSpace(strings.ToLower(params.Email))

	var validation fieldValidationBuilder
	validation.addRequired("email", params.Email, "Email")
	if params.Email != "" {
		email := params.Email
		if err := validateEmail(&email); err != nil {
			validation.add("email", "invalid_email", "Email указан неверно")
		}
	}

	if err := validation.err(); err != nil {
		return domain.ForgotPasswordParams{}, err
	}

	return params, nil
}

func normalizeResetPasswordParams(params domain.ResetPasswordParams) (domain.ResetPasswordParams, error) {
	params.Token = strings.TrimSpace(params.Token)
	params.NewPassword = strings.TrimSpace(params.NewPassword)

	var validation fieldValidationBuilder
	validation.addRequired("token", params.Token, "Токен сброса")
	validation.addRequired("new_password", params.NewPassword, "Новый пароль")
	if params.NewPassword != "" && len(params.NewPassword) < 8 {
		validation.add("new_password", "too_short", "Новый пароль должен быть минимум 8 символов")
	}

	if err := validation.err(); err != nil {
		return domain.ResetPasswordParams{}, err
	}

	return params, nil
}

func profileUpdateToUserParams(current domain.User, params domain.UpdateProfileParams) domain.UpdateUserParams {
	studentInfo := current.StudentInfo
	if current.Role == domain.UserRoleStudent {
		if studentInfo == nil {
			studentInfo = &domain.StudentInfo{}
		}
		copied := *studentInfo
		if params.BirthDate != nil {
			copied.BirthDate = *params.BirthDate
		}
		studentInfo = &copied
	}

	return domain.UpdateUserParams{
		ID:           current.ID,
		Email:        current.Email,
		GoogleID:     current.GoogleID,
		Password:     nil,
		Role:         current.Role,
		FirstName:    keepExistingString(params.FirstName, current.FirstName),
		LastName:     keepExistingString(params.LastName, current.LastName),
		Patronymic:   current.Patronymic,
		Phone:        keepExistingOptionalString(params.Phone, current.Phone),
		Gender:       current.Gender,
		BirthDate:    keepExistingOptionalString(params.BirthDate, current.BirthDate),
		Address:      current.Address,
		City:         current.City,
		AvatarURL:    current.AvatarURL,
		IsActive:     current.IsActive,
		EmployeeInfo: current.EmployeeInfo,
		StudentInfo:  studentInfo,
		GuestInfo:    current.GuestInfo,
	}
}

func userToPasswordUpdateParams(user domain.User) domain.UpdateUserParams {
	return domain.UpdateUserParams{
		ID:           user.ID,
		Email:        user.Email,
		GoogleID:     user.GoogleID,
		Role:         user.Role,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		Patronymic:   user.Patronymic,
		Phone:        user.Phone,
		Gender:       user.Gender,
		BirthDate:    user.BirthDate,
		Address:      user.Address,
		City:         user.City,
		AvatarURL:    user.AvatarURL,
		IsActive:     user.IsActive,
		EmployeeInfo: user.EmployeeInfo,
		StudentInfo:  user.StudentInfo,
		GuestInfo:    user.GuestInfo,
	}
}

func keepExistingString(next, current string) string {
	if strings.TrimSpace(next) == "" {
		return current
	}
	return next
}

func keepExistingOptionalString(next, current *string) *string {
	if next == nil {
		return current
	}
	return next
}

func genderFromBoolean(current domain.Gender, isMale *bool) domain.Gender {
	if isMale == nil {
		return current
	}

	if *isMale {
		return domain.GenderMale
	}

	return domain.GenderFemale
}

func stringFromPointer(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func generateSessionToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("usecase auth generate session token: %w", err)
	}

	return hex.EncodeToString(buffer), nil
}

func hashPasswordResetToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

func (u *AuthUseCase) ensureLoginAllowed(ctx context.Context, identifier string, ipAddress *string) error {
	if !u.loginLockoutEnabled || u.loginAttempts == nil || u.maxLoginAttempts <= 0 || u.loginAttemptWindow <= 0 {
		return nil
	}

	count, err := u.loginAttempts.CountRecentFailed(ctx, identifier, ipAddress, u.loginLockoutScope, u.now().UTC().Add(-u.loginAttemptWindow))
	if err != nil {
		return fmt.Errorf("count recent failed attempts: %w", err)
	}

	if count >= u.maxLoginAttempts {
		return domain.TooManyRequestsError("too_many_attempts", "Слишком много попыток входа. Попробуйте позже")
	}

	return nil
}

func normalizeLoginLockoutScope(scope string) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "identifier", "ip", "identifier_ip":
		return strings.ToLower(strings.TrimSpace(scope))
	default:
		return "identifier_ip"
	}
}

func (u *AuthUseCase) recordLoginAttempt(ctx context.Context, identifier string, ipAddress *string, succeeded *bool) {
	if !u.loginLockoutEnabled || u.loginAttempts == nil || succeeded == nil {
		return
	}

	_ = u.loginAttempts.Record(ctx, identifier, ipAddress, *succeeded)
}
