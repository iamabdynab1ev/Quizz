package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"request-system/internal/dto"
	"request-system/internal/entities"
	"request-system/internal/repositories"
	"request-system/pkg/config"
	pkgconstants "request-system/pkg/constants"
	apperrors "request-system/pkg/errors"
	"request-system/pkg/utils"

	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type authTxManagerStub struct{}

func (authTxManagerStub) RunInTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
	return fn(nil)
}

type authUserRepoStub struct {
	repositories.UserRepositoryInterface
	user  *entities.User
	err   error
	calls int
}

func (r *authUserRepoStub) FindUserByEmailOrLogin(ctx context.Context, login string) (*entities.User, error) {
	r.calls++
	if r.err != nil {
		return nil, r.err
	}
	if r.user == nil {
		return nil, pgx.ErrNoRows
	}

	user := *r.user
	return &user, nil
}

type authCacheStub struct {
	mu     sync.Mutex
	values map[string]string
}

func newAuthCacheStub() *authCacheStub {
	return &authCacheStub{values: map[string]string{}}
}

func (c *authCacheStub) Get(ctx context.Context, key string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	value, ok := c.values[key]
	if !ok {
		return "", redis.Nil
	}
	return value, nil
}

func (c *authCacheStub) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.values[key] = fmt.Sprint(value)
	return nil
}

func (c *authCacheStub) Del(ctx context.Context, keys ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, key := range keys {
		delete(c.values, key)
	}
	return nil
}

func (c *authCacheStub) Incr(ctx context.Context, key string) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	current := int64(0)
	if raw, ok := c.values[key]; ok && raw != "" {
		parsed, err := parseInt64(raw)
		if err != nil {
			return 0, err
		}
		current = parsed
	}

	current++
	c.values[key] = fmt.Sprint(current)
	return current, nil
}

func (c *authCacheStub) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	return true, nil
}

func parseInt64(value string) (int64, error) {
	var parsed int64
	_, err := fmt.Sscan(value, &parsed)
	return parsed, err
}

func newAuthServiceForTest(userRepo *authUserRepoStub, cache *authCacheStub, cfg *config.AuthConfig) *AuthService {
	return &AuthService{
		txManager: authTxManagerStub{},
		userRepo:  userRepo,
		cacheRepo: cache,
		logger:    zap.NewNop(),
		cfg:       cfg,
		ldapCfg:   &config.LDAPConfig{Enabled: false},
	}
}

func TestAuthLogin_LocksAccountAfterMaxFailures(t *testing.T) {
	cache := newAuthCacheStub()
	passwordHash, err := utils.HashPassword("correct-password")
	require.NoError(t, err)

	userRepo := &authUserRepoStub{
		user: &entities.User{
			ID:                 42,
			Email:              "admin@example.com",
			Password:           passwordHash,
			StatusCode:         pkgconstants.UserStatusActiveCode,
			Username:           strPtr("admin"),
			MustChangePassword: false,
		},
	}
	svc := newAuthServiceForTest(userRepo, cache, &config.AuthConfig{
		MaxLoginAttempts: 3,
		LockoutDuration:  10 * time.Minute,
	})

	payload := dto.LoginDTO{Login: "admin@example.com", Password: "wrong-password"}

	for i := 0; i < 2; i++ {
		_, err := svc.Login(context.Background(), payload)
		require.Error(t, err)
		var httpErr *apperrors.HttpError
		require.True(t, errors.As(err, &httpErr))
		require.Equal(t, 401, httpErr.Code)
	}

	_, err = svc.Login(context.Background(), payload)
	require.Error(t, err)
	var httpErr *apperrors.HttpError
	require.True(t, errors.As(err, &httpErr))
	require.Equal(t, 403, httpErr.Code)
	require.Equal(t, apperrors.ErrAccountLocked.Message, httpErr.Message)

	require.Equal(t, "3", cache.values[fmt.Sprintf(pkgconstants.CacheKeyLoginAttemptsByLogin, "admin@example.com")])
	require.Equal(t, "3", cache.values[fmt.Sprintf(pkgconstants.CacheKeyLoginAttempts, uint64(42))])
	require.Equal(t, "locked", cache.values[fmt.Sprintf(pkgconstants.CacheKeyLockoutByLogin, "admin@example.com")])
	require.Equal(t, "locked", cache.values[fmt.Sprintf(pkgconstants.CacheKeyLockout, uint64(42))])
}

func TestAuthLogin_RejectsAlreadyLockedAccountBeforeRepoLookup(t *testing.T) {
	cache := newAuthCacheStub()
	cache.values[fmt.Sprintf(pkgconstants.CacheKeyLockoutByLogin, "admin@example.com")] = "locked"

	userRepo := &authUserRepoStub{
		user: &entities.User{
			ID:         42,
			Email:      "admin@example.com",
			Password:   "hash",
			StatusCode: pkgconstants.UserStatusActiveCode,
		},
	}
	svc := newAuthServiceForTest(userRepo, cache, &config.AuthConfig{
		MaxLoginAttempts: 3,
		LockoutDuration:  10 * time.Minute,
	})

	_, err := svc.Login(context.Background(), dto.LoginDTO{Login: "admin@example.com", Password: "correct-password"})
	require.Error(t, err)

	var httpErr *apperrors.HttpError
	require.True(t, errors.As(err, &httpErr))
	require.Equal(t, 403, httpErr.Code)
	require.Equal(t, 0, userRepo.calls)
}

func TestAuthLogin_ClearsLockoutStateOnSuccess(t *testing.T) {
	cache := newAuthCacheStub()
	login := "admin@example.com"
	userID := uint64(42)

	cache.values[fmt.Sprintf(pkgconstants.CacheKeyLoginAttemptsByLogin, login)] = "2"
	cache.values[fmt.Sprintf(pkgconstants.CacheKeyLoginAttempts, userID)] = "2"

	passwordHash, err := utils.HashPassword("correct-password")
	require.NoError(t, err)

	userRepo := &authUserRepoStub{
		user: &entities.User{
			ID:         userID,
			Email:      login,
			Password:   passwordHash,
			StatusCode: pkgconstants.UserStatusActiveCode,
			Username:   strPtr("admin"),
		},
	}
	svc := newAuthServiceForTest(userRepo, cache, &config.AuthConfig{
		MaxLoginAttempts: 3,
		LockoutDuration:  10 * time.Minute,
	})

	_, err = svc.Login(context.Background(), dto.LoginDTO{Login: login, Password: "correct-password"})
	require.NoError(t, err)

	_, exists := cache.values[fmt.Sprintf(pkgconstants.CacheKeyLoginAttemptsByLogin, login)]
	require.False(t, exists)
	_, exists = cache.values[fmt.Sprintf(pkgconstants.CacheKeyLoginAttempts, userID)]
	require.False(t, exists)
	_, exists = cache.values[fmt.Sprintf(pkgconstants.CacheKeyLockoutByLogin, login)]
	require.False(t, exists)
	_, exists = cache.values[fmt.Sprintf(pkgconstants.CacheKeyLockout, userID)]
	require.False(t, exists)
}

func TestAuthRefreshSessionRegistry_RoundTripsAndInvalidates(t *testing.T) {
	cache := newAuthCacheStub()
	svc := newAuthServiceForTest(&authUserRepoStub{}, cache, &config.AuthConfig{})

	ctx := context.Background()
	userID := uint64(42)
	sessionID := "session-1"

	require.NoError(t, svc.RegisterRefreshSession(ctx, userID, sessionID, time.Hour))
	require.NoError(t, svc.ValidateRefreshSession(ctx, userID, sessionID))
	require.Error(t, svc.ValidateRefreshSession(ctx, userID+1, sessionID))

	require.NoError(t, svc.InvalidateRefreshSession(ctx, sessionID))
	require.Error(t, svc.ValidateRefreshSession(ctx, userID, sessionID))
}

func TestAuthRefreshSessionRegistry_RejectsEmptySessionID(t *testing.T) {
	cache := newAuthCacheStub()
	svc := newAuthServiceForTest(&authUserRepoStub{}, cache, &config.AuthConfig{})

	ctx := context.Background()

	require.Error(t, svc.RegisterRefreshSession(ctx, 42, "", time.Hour))
	require.Error(t, svc.ValidateRefreshSession(ctx, 42, ""))
	require.NoError(t, svc.InvalidateRefreshSession(ctx, ""))
}

func strPtr(v string) *string {
	return &v
}
