package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIKeyStorageValueDoesNotRetainSecret(t *testing.T) {
	secret := "sk-abcdefghijklmnopqrstuvwxyz0123456789"
	stored := HashAPIKeyForStorage(secret)

	require.NotEqual(t, secret, stored)
	require.NotContains(t, stored, "ghijklmnopqrstuvwxyz")
	require.Equal(t, stored, HashAPIKeyForStorage(stored))
	require.Equal(t, "sk-abc...6789", MaskAPIKeyForDisplay(stored))
	require.Equal(t, APIKeyCacheDigest(secret), APIKeyCacheDigest(stored))
}

func TestAPIKeyLookupValuesKeepLegacyFallback(t *testing.T) {
	secret := "sk-legacy-1234567890"
	values := APIKeyLookupValues(secret)

	require.Len(t, values, 2)
	require.Equal(t, HashAPIKeyForStorage(secret), values[0])
	require.Equal(t, secret, values[1])
}

func TestAPIKeyStorageValueHandlesConfiguredDelimiter(t *testing.T) {
	secret := "sk$custom-prefix-abcdefghijklmnopqrstuvwxyz"
	stored := HashAPIKeyForStorage(secret)

	require.Equal(t, "sk$cus...wxyz", MaskAPIKeyForDisplay(stored))
	require.Equal(t, APIKeyCacheDigest(secret), APIKeyCacheDigest(stored))
}
