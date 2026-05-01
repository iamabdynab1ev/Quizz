package cache

import (
	"testing"
	"time"

	"lms-arvand-backend/internal/domain"
)

func TestSessionCacheSetGetAndDelete(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	cache := NewSessionCache(5 * time.Minute)
	cache.now = func() time.Time { return now }

	identity := domain.AuthIdentity{
		User: domain.User{ID: "user-1"},
		Session: domain.Session{
			Token:     "token-1",
			UserID:    "user-1",
			ExpiresAt: ptrTime(now.Add(10 * time.Minute)),
		},
	}

	cache.Set(identity.Session.Token, identity, identity.Session.ExpiresAt)

	got, ok := cache.Get(identity.Session.Token)
	if !ok {
		t.Fatalf("expected cache hit")
	}
	if got.User.ID != identity.User.ID {
		t.Fatalf("unexpected user id: %s", got.User.ID)
	}

	cache.Delete(identity.Session.Token)
	if _, ok := cache.Get(identity.Session.Token); ok {
		t.Fatalf("expected cache miss after delete")
	}
}

func TestSessionCacheExpiresByTTL(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	cache := NewSessionCache(1 * time.Minute)
	cache.now = func() time.Time { return now }

	identity := domain.AuthIdentity{
		User: domain.User{ID: "user-2"},
		Session: domain.Session{
			Token:     "token-2",
			UserID:    "user-2",
			ExpiresAt: ptrTime(now.Add(30 * time.Minute)),
		},
	}

	cache.Set(identity.Session.Token, identity, identity.Session.ExpiresAt)

	cache.now = func() time.Time { return now.Add(2 * time.Minute) }
	if _, ok := cache.Get(identity.Session.Token); ok {
		t.Fatalf("expected cache entry to expire by ttl")
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
