package service

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
)

const (
	apiKeyStoragePrefix    = "sha256$"
	apiKeyClientHashPrefix = "sk-hash-"
)

// HashAPIKeyForStorage returns a one-way database representation while retaining
// only a short prefix and suffix for identification in the UI.
func HashAPIKeyForStorage(key string) string {
	if _, _, _, ok := parseHashedAPIKey(key); ok {
		return key
	}
	if stored, _, ok := parseClientHashedAPIKey(key); ok {
		return stored
	}
	sum := sha256.Sum256([]byte(key))
	encodedPrefix := hex.EncodeToString([]byte(apiKeyVisiblePrefix(key)))
	encodedSuffix := hex.EncodeToString([]byte(apiKeyVisibleSuffix(key)))
	return apiKeyStoragePrefix + hex.EncodeToString(sum[:]) + "$" + encodedPrefix + "$" + encodedSuffix
}

// APIKeyLookupValues supports databases that have not yet applied the hashing
// migration. The hashed value is always preferred.
func APIKeyLookupValues(key string) []string {
	if stored, _, ok := parseClientHashedAPIKey(key); ok {
		return []string{stored, key}
	}
	hashed := HashAPIKeyForStorage(key)
	if hashed == key {
		return []string{key}
	}
	return []string{hashed, key}
}

// APIKeyForClient converts a migrated database hash to an API-key-safe bearer
// token. The token contains the same verifier data but avoids '$', which can be
// altered by some proxy and client configurations. Plaintext keys are unchanged.
func APIKeyForClient(key string) string {
	digest, prefix, suffix, ok := parseHashedAPIKey(key)
	if !ok {
		return key
	}
	return apiKeyClientHashPrefix + digest + "-" +
		hex.EncodeToString([]byte(prefix)) + "-" +
		hex.EncodeToString([]byte(suffix))
}

// MaskAPIKeyForDisplay never returns a usable secret for stored API key values.
func MaskAPIKeyForDisplay(key string) string {
	_, prefix, suffix, ok := parseHashedAPIKey(key)
	if !ok {
		prefix = apiKeyVisiblePrefix(key)
		suffix = apiKeyVisibleSuffix(key)
	}
	if prefix == "" {
		return ""
	}
	return prefix + "..." + suffix
}

// APIKeyCacheDigest returns the authentication cache key for either a plaintext
// secret or its stored representation.
func APIKeyCacheDigest(key string) string {
	if digest, _, _, ok := parseHashedAPIKey(key); ok {
		return digest
	}
	if _, digest, ok := parseClientHashedAPIKey(key); ok {
		return digest
	}
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func parseClientHashedAPIKey(value string) (stored, digest string, ok bool) {
	if !strings.HasPrefix(value, apiKeyClientHashPrefix) {
		return "", "", false
	}
	parts := strings.Split(strings.TrimPrefix(value, apiKeyClientHashPrefix), "-")
	if len(parts) != 3 {
		return "", "", false
	}
	stored = apiKeyStoragePrefix + parts[0] + "$" + parts[1] + "$" + parts[2]
	digest, _, _, ok = parseHashedAPIKey(stored)
	if !ok {
		return "", "", false
	}
	return stored, digest, true
}

func parseHashedAPIKey(value string) (digest, prefix, suffix string, ok bool) {
	if !strings.HasPrefix(value, apiKeyStoragePrefix) {
		return "", "", "", false
	}
	parts := strings.Split(value, "$")
	if len(parts) != 4 || parts[0] != "sha256" || len(parts[1]) != sha256.Size*2 {
		return "", "", "", false
	}
	if _, err := hex.DecodeString(parts[1]); err != nil {
		return "", "", "", false
	}
	prefixBytes, err := hex.DecodeString(parts[2])
	if err != nil || len(prefixBytes) > 6 {
		return "", "", "", false
	}
	suffixBytes, err := hex.DecodeString(parts[3])
	if err != nil || len(suffixBytes) > 4 {
		return "", "", "", false
	}
	return parts[1], string(prefixBytes), string(suffixBytes), true
}

func apiKeyVisiblePrefix(key string) string {
	if len(key) <= 6 {
		return key
	}
	return key[:6]
}

func apiKeyVisibleSuffix(key string) string {
	if len(key) <= 4 {
		return ""
	}
	return key[len(key)-4:]
}

// API Key status constants
const (
	StatusAPIKeyActive         = "active"
	StatusAPIKeyDisabled       = "disabled"
	StatusAPIKeyQuotaExhausted = "quota_exhausted"
	StatusAPIKeyExpired        = "expired"
)

// Rate limit window durations
const (
	RateLimitWindow5h = 5 * time.Hour
	RateLimitWindow1d = 24 * time.Hour
	RateLimitWindow7d = 7 * 24 * time.Hour
)

// IsWindowExpired returns true if the window starting at windowStart has exceeded the given duration.
// A nil windowStart is treated as expired — no initialized window means any accumulated usage is stale.
func IsWindowExpired(windowStart *time.Time, duration time.Duration) bool {
	return windowStart == nil || time.Since(*windowStart) >= duration
}

type APIKey struct {
	ID          int64
	UserID      int64
	Key         string
	Name        string
	GroupID     *int64
	Status      string
	IPWhitelist []string
	IPBlacklist []string
	// 预编译的 IP 规则，用于认证热路径避免重复 ParseIP/ParseCIDR。
	CompiledIPWhitelist *ip.CompiledIPRules `json:"-"`
	CompiledIPBlacklist *ip.CompiledIPRules `json:"-"`
	LastUsedAt          *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
	User                *User
	Group               *Group

	// Quota fields
	Quota     float64    // Quota limit in USD (0 = unlimited)
	QuotaUsed float64    // Used quota amount
	ExpiresAt *time.Time // Expiration time (nil = never expires)

	// Rate limit fields
	RateLimit5h   float64    // Rate limit in USD per 5h (0 = unlimited)
	RateLimit1d   float64    // Rate limit in USD per 1d (0 = unlimited)
	RateLimit7d   float64    // Rate limit in USD per 7d (0 = unlimited)
	Usage5h       float64    // Used amount in current 5h window
	Usage1d       float64    // Used amount in current 1d window
	Usage7d       float64    // Used amount in current 7d window
	Window5hStart *time.Time // Start of current 5h window
	Window1dStart *time.Time // Start of current 1d window
	Window7dStart *time.Time // Start of current 7d window
}

func (k *APIKey) IsActive() bool {
	return k.Status == StatusActive
}

// HasRateLimits returns true if any rate limit window is configured
func (k *APIKey) HasRateLimits() bool {
	return k.RateLimit5h > 0 || k.RateLimit1d > 0 || k.RateLimit7d > 0
}

// IsExpired checks if the API key has expired
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// IsQuotaExhausted checks if the API key quota is exhausted
func (k *APIKey) IsQuotaExhausted() bool {
	if k.Quota <= 0 {
		return false // unlimited
	}
	return k.QuotaUsed >= k.Quota
}

// GetQuotaRemaining returns remaining quota (-1 for unlimited)
func (k *APIKey) GetQuotaRemaining() float64 {
	if k.Quota <= 0 {
		return -1 // unlimited
	}
	remaining := k.Quota - k.QuotaUsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetDaysUntilExpiry returns days until expiry (-1 for never expires)
func (k *APIKey) GetDaysUntilExpiry() int {
	if k.ExpiresAt == nil {
		return -1 // never expires
	}
	duration := time.Until(*k.ExpiresAt)
	if duration < 0 {
		return 0
	}
	return int(duration.Hours() / 24)
}

// EffectiveUsage5h returns the 5h window usage, or 0 if the window has expired.
func (k *APIKey) EffectiveUsage5h() float64 {
	if IsWindowExpired(k.Window5hStart, RateLimitWindow5h) {
		return 0
	}
	return k.Usage5h
}

// EffectiveUsage1d returns the 1d window usage, or 0 if the window has expired.
func (k *APIKey) EffectiveUsage1d() float64 {
	if IsWindowExpired(k.Window1dStart, RateLimitWindow1d) {
		return 0
	}
	return k.Usage1d
}

// EffectiveUsage7d returns the 7d window usage, or 0 if the window has expired.
func (k *APIKey) EffectiveUsage7d() float64 {
	if IsWindowExpired(k.Window7dStart, RateLimitWindow7d) {
		return 0
	}
	return k.Usage7d
}

// APIKeyListFilters holds optional filtering parameters for listing API keys.
type APIKeyListFilters struct {
	Search  string
	Status  string
	GroupID *int64 // nil=不筛选, 0=无分组, >0=指定分组
}
