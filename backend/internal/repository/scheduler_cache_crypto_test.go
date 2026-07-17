package repository

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newEncryptedSchedulerCache(t *testing.T) (*schedulerCache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	cache := NewSchedulerCache(rdb, accountCredentialTestEncryptor{}).(*schedulerCache)
	return cache, mr
}

func TestSchedulerCacheEncryptsCredentialsAtRest(t *testing.T) {
	cache, mr := newEncryptedSchedulerCache(t)
	ctx := context.Background()
	account := &service.Account{
		ID:       42,
		Name:     "encrypted-cache-account",
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":      "sk-upstream-secret",
			"access_token": "oauth-access-secret",
		},
	}

	require.NoError(t, cache.SetAccount(ctx, account))
	raw, err := mr.Get(schedulerAccountKey(strconv.FormatInt(account.ID, 10)))
	require.NoError(t, err)
	require.NotContains(t, raw, "sk-upstream-secret")
	require.NotContains(t, raw, "oauth-access-secret")
	require.Contains(t, raw, accountCredentialsCiphertextKey)

	got, err := cache.GetAccount(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, "sk-upstream-secret", got.GetCredential("api_key"))
	require.Equal(t, "oauth-access-secret", got.GetCredential("access_token"))
}

func TestSchedulerCacheMigratesLegacyPlaintextCredentials(t *testing.T) {
	cache, mr := newEncryptedSchedulerCache(t)
	ctx := context.Background()
	legacy := service.Account{
		ID:          7,
		Credentials: map[string]any{"api_key": "legacy-upstream-secret"},
	}
	payload, err := json.Marshal(legacy)
	require.NoError(t, err)
	key := schedulerAccountKey(strconv.FormatInt(legacy.ID, 10))
	mr.Set(key, string(payload))

	require.NoError(t, cache.migrateCredentialsAtRest(ctx))
	raw, err := mr.Get(key)
	require.NoError(t, err)
	require.NotContains(t, raw, "legacy-upstream-secret")
	require.Contains(t, raw, accountCredentialsCiphertextKey)

	got, err := cache.GetAccount(ctx, legacy.ID)
	require.NoError(t, err)
	require.Equal(t, "legacy-upstream-secret", got.GetCredential("api_key"))
}
