package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"lms-arvand-backend/internal/domain"
)

type authUserRepository interface {
	GetByLogin(ctx context.Context, identifier string) (domain.User, error)
	GetByID(ctx context.Context, userID string) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	GetByGoogleID(ctx context.Context, googleID string) (domain.User, error)
	LinkGoogleID(ctx context.Context, userID, googleID string) (domain.User, error)
	Create(ctx context.Context, params domain.CreateUserParams) (domain.User, error)
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
	CountRecentFailed(ctx context.Context, identifier string, ipAddress *string, since time.Time) (int, error)
	Record(ctx context.Context, identifier string, ipAddress *string, succeeded bool) error
}

type AuthUseCase struct {
	users              authUserRepository
	sessions           authSessionRepository
	sessionCache       authSessionCache
	loginAttempts      authLoginAttemptRepository
	sessionTTL         time.Duration
	maxLoginAttempts   int
	loginAttemptWindow time.Duration
	google             googleTokenVerifier
	now                func() time.Time
}

func NewAuthUseCase(
	users authUserRepository,
	sessions authSessionRepository,
	sessionCache authSessionCache,
	sessionTTL time.Duration,
	loginAttempts authLoginAttemptRepository,
	maxLoginAttempts int,
	loginAttemptWindow time.Duration,
	google googleTokenVerifier,
) *AuthUseCase {
	return &AuthUseCase{
		users:              users,
		sessions:           sessions,
		sessionCache:       sessionCache,
		loginAttempts:      loginAttempts,
		sessionTTL:         sessionTTL,
		maxLoginAttempts:   maxLoginAttempts,
		loginAttemptWindow: loginAttemptWindow,
		google:             google,
		now:                time.Now,
	}
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

	user, err := u.users.GetByLogin(ctx, normalized.Identifier)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.LoginResult{}, fmt.Errorf("usecase auth login: %w", domain.ErrUnauthorized)
		}

		return domain.LoginResult{}, fmt.Errorf("usecase auth login get user: %w", err)
	}

	if !user.IsActive {
		return domain.LoginResult{}, fmt.Errorf("usecase auth login inactive user: %w", domain.ErrUnauthorized)
	}

	if user.PasswordHash == nil {
		return domain.LoginResult{}, fmt.Errorf("usecase auth login missing password hash: %w", domain.ErrUnauthorized)
	}

	if err := comparePasswordHash(*user.PasswordHash, normalized.Password); err != nil {
		return domain.LoginResult{}, fmt.Errorf("usecase auth login compare password: %w", domain.ErrUnauthorized)
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
		return domain.LoginResult{}, fmt.Errorf("usecase auth google login: %w", domain.UnavailableError("google sign-in is not configured"))
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
			return domain.User{}, domain.ConflictError("google account is already linked to another identity")
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

	username := generateGoogleUsername(identity.Email, identity.Subject)
	googleID := identity.Subject
	email := identity.Email
	avatarURL := normalizeOptionalString(&identity.Picture)

	createdUser, err := u.users.Create(ctx, domain.CreateUserParams{
		Username:  username,
		Email:     &email,
		GoogleID:  &googleID,
		Role:      domain.UserRoleGuest,
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
		return domain.GoogleLoginParams{}, fmt.Errorf("id_token is required: %w", domain.ErrValidation)
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

func generateGoogleUsername(email, subject string) string {
	base := email
	if localPart, _, found := strings.Cut(email, "@"); found {
		base = localPart
	}

	base = strings.ToLower(strings.TrimSpace(base))
	if base == "" {
		base = "google_user"
	}

	var builder strings.Builder
	for _, symbol := range base {
		switch {
		case symbol >= 'a' && symbol <= 'z':
			builder.WriteRune(symbol)
		case symbol >= '0' && symbol <= '9':
			builder.WriteRune(symbol)
		case symbol == '_' || symbol == '-' || symbol == '.':
			builder.WriteRune(symbol)
		default:
			builder.WriteByte('_')
		}
	}

	sanitized := strings.Trim(builder.String(), "_.-")
	if sanitized == "" {
		sanitized = "google_user"
	}

	suffix := strings.TrimSpace(subject)
	if len(suffix) > 8 {
		suffix = suffix[len(suffix)-8:]
	}

	if suffix == "" {
		return sanitized
	}

	username := sanitized + "_" + strings.ToLower(suffix)
	if len(username) > 64 {
		username = username[:64]
	}

	return username
}

func normalizeLoginParams(params domain.LoginParams) (domain.LoginParams, error) {
	params.Identifier = strings.TrimSpace(params.Identifier)
	params.Password = strings.TrimSpace(params.Password)
	params.IPAddress = normalizeOptionalString(params.IPAddress)
	params.UserAgent = normalizeOptionalString(params.UserAgent)

	if params.Identifier == "" {
		return domain.LoginParams{}, fmt.Errorf("identifier is required: %w", domain.ErrValidation)
	}

	if params.Password == "" {
		return domain.LoginParams{}, fmt.Errorf("password is required: %w", domain.ErrValidation)
	}

	return params, nil
}

func generateSessionToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("usecase auth generate session token: %w", err)
	}

	return hex.EncodeToString(buffer), nil
}

func (u *AuthUseCase) ensureLoginAllowed(ctx context.Context, identifier string, ipAddress *string) error {
	if u.loginAttempts == nil || u.maxLoginAttempts <= 0 || u.loginAttemptWindow <= 0 {
		return nil
	}

	count, err := u.loginAttempts.CountRecentFailed(ctx, identifier, ipAddress, u.now().UTC().Add(-u.loginAttemptWindow))
	if err != nil {
		return fmt.Errorf("count recent failed attempts: %w", err)
	}

	if count >= u.maxLoginAttempts {
		return domain.TooManyRequestsError("too_many_attempts", "too many login attempts, try again later")
	}

	return nil
}

func (u *AuthUseCase) recordLoginAttempt(ctx context.Context, identifier string, ipAddress *string, succeeded *bool) {
	if u.loginAttempts == nil || succeeded == nil {
		return
	}

	_ = u.loginAttempts.Record(ctx, identifier, ipAddress, *succeeded)
}
