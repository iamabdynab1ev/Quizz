package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"request-system/internal/repositories"
)

const smallDictionaryCacheTTL = 5 * time.Minute

func readVersionedListCache[T any](ctx context.Context, cache repositories.CacheRepositoryInterface, namespace string, keyPayload any) (T, bool) {
	var zero T
	if cache == nil {
		return zero, false
	}

	cacheKey, err := buildVersionedListCacheKey(ctx, cache, namespace, keyPayload)
	if err != nil {
		return zero, false
	}

	cached, err := cache.Get(ctx, cacheKey)
	if err != nil {
		return zero, false
	}

	var value T
	if err := json.Unmarshal([]byte(cached), &value); err != nil {
		return zero, false
	}

	return value, true
}

func writeVersionedListCache[T any](ctx context.Context, cache repositories.CacheRepositoryInterface, namespace string, keyPayload any, value T) {
	if cache == nil {
		return
	}

	cacheKey, err := buildVersionedListCacheKey(ctx, cache, namespace, keyPayload)
	if err != nil {
		return
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return
	}

	_ = cache.Set(ctx, cacheKey, payload, smallDictionaryCacheTTL)
}

func invalidateVersionedListCache(ctx context.Context, cache repositories.CacheRepositoryInterface, namespace string) {
	if cache == nil {
		return
	}

	versionKey := namespace + ":version"
	if _, err := cache.Incr(ctx, versionKey); err == nil {
		_, _ = cache.Expire(ctx, versionKey, 24*time.Hour)
	}
}

func buildVersionedListCacheKey(ctx context.Context, cache repositories.CacheRepositoryInterface, namespace string, keyPayload any) (string, error) {
	payload, err := json.Marshal(keyPayload)
	if err != nil {
		return "", err
	}

	version := "0"
	if cachedVersion, err := cache.Get(ctx, namespace+":version"); err == nil && strings.TrimSpace(cachedVersion) != "" {
		version = strings.TrimSpace(cachedVersion)
	}

	sum := sha256.Sum256(payload)
	return fmt.Sprintf("%s:v%s:%s", namespace, version, hex.EncodeToString(sum[:])), nil
}
