package dto

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyFromService_MapsLastUsedAt(t *testing.T) {
	lastUsed := time.Now().UTC().Truncate(time.Second)
	src := &service.APIKey{
		ID:         1,
		UserID:     2,
		Key:        "sk-map-last-used",
		Name:       "Mapper",
		Status:     service.StatusActive,
		LastUsedAt: &lastUsed,
	}

	out := APIKeyFromService(src)
	require.NotNil(t, out)
	require.NotNil(t, out.LastUsedAt)
	require.WithinDuration(t, lastUsed, *out.LastUsedAt, time.Second)
}

func TestAPIKeyFromService_MapsNilLastUsedAt(t *testing.T) {
	src := &service.APIKey{
		ID:     1,
		UserID: 2,
		Key:    "sk-map-last-used-nil",
		Name:   "MapperNil",
		Status: service.StatusActive,
	}

	out := APIKeyFromService(src)
	require.NotNil(t, out)
	require.Nil(t, out.LastUsedAt)
}

func TestAPIKeyFromService_RevealsSecretForCopying(t *testing.T) {
	src := &service.APIKey{Key: "sk-abcdefghijklmnopqrstuvwxyz0123456789"}

	listed := APIKeyFromService(src)
	require.Equal(t, src.Key, listed.Key)
	require.True(t, listed.KeyRevealed)

	created := APIKeyFromServiceWithSecret(src)
	require.Equal(t, src.Key, created.Key)
	require.True(t, created.KeyRevealed)
}

func TestAPIKeyFromService_ConvertsMigratedHashToClientToken(t *testing.T) {
	stored := service.HashAPIKeyForStorage("sk-migrated-abcdefghijklmnopqrstuvwxyz")

	listed := APIKeyFromService(&service.APIKey{Key: stored})

	require.Equal(t, service.APIKeyForClient(stored), listed.Key)
	require.NotContains(t, listed.Key, "$")
	require.True(t, listed.KeyRevealed)
}
