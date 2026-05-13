package cache

import (
	"sync"
	"time"

	"lms-arvand-backend/internal/domain"
)

type SessionCache struct {
	mu    sync.RWMutex
	items map[string]sessionCacheEntry
	ttl   time.Duration
	now   func() time.Time
}

const evictionInterval = 5 * time.Minute

type sessionCacheEntry struct {
	identity  domain.AuthIdentity
	expiresAt *time.Time
}

func NewSessionCache(ttl time.Duration) *SessionCache {
	c := &SessionCache{
		items: make(map[string]sessionCacheEntry),
		ttl:   ttl,
		now:   time.Now,
	}
	go c.runEviction()
	return c
}

func (c *SessionCache) runEviction() {
	ticker := time.NewTicker(evictionInterval)
	defer ticker.Stop()
	for range ticker.C {
		now := c.now().UTC()
		c.mu.Lock()
		for token, entry := range c.items {
			if entry.expiresAt != nil && !entry.expiresAt.After(now) {
				delete(c.items, token)
			}
		}
		c.mu.Unlock()
	}
}

func (c *SessionCache) Get(token string) (domain.AuthIdentity, bool) {
	if c == nil {
		return domain.AuthIdentity{}, false
	}

	now := c.now().UTC()

	c.mu.RLock()
	entry, ok := c.items[token]
	c.mu.RUnlock()
	if !ok {
		return domain.AuthIdentity{}, false
	}

	if entry.expiresAt != nil && !entry.expiresAt.After(now) {
		c.mu.Lock()
		delete(c.items, token)
		c.mu.Unlock()
		return domain.AuthIdentity{}, false
	}

	return entry.identity, true
}

func (c *SessionCache) Set(token string, identity domain.AuthIdentity, sessionExpiresAt *time.Time) {
	if c == nil || token == "" {
		return
	}

	var expiresAt *time.Time
	now := c.now().UTC()

	switch {
	case c.ttl > 0 && sessionExpiresAt != nil:
		cacheExpiresAt := now.Add(c.ttl)
		if sessionExpiresAt.Before(cacheExpiresAt) {
			expiresAt = cloneTimePtr(sessionExpiresAt)
		} else {
			expiresAt = &cacheExpiresAt
		}
	case c.ttl > 0:
		cacheExpiresAt := now.Add(c.ttl)
		expiresAt = &cacheExpiresAt
	case sessionExpiresAt != nil:
		expiresAt = cloneTimePtr(sessionExpiresAt)
	}

	c.mu.Lock()
	c.items[token] = sessionCacheEntry{
		identity:  identity,
		expiresAt: expiresAt,
	}
	c.mu.Unlock()
}

func (c *SessionCache) Delete(token string) {
	if c == nil || token == "" {
		return
	}

	c.mu.Lock()
	delete(c.items, token)
	c.mu.Unlock()
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	cloned := value.UTC()
	return &cloned
}
