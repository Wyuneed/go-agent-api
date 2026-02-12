package entity_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
)

func TestNewAPIKey(t *testing.T) {
	userID := uuid.New()
	expiresAt := time.Now().Add(24 * time.Hour)

	token := entity.NewAPIKey(userID, "hash123", "My API Key", expiresAt)

	assert.NotNil(t, token)
	assert.Equal(t, userID, token.UserID)
	assert.Equal(t, "hash123", token.TokenHash)
	assert.Equal(t, entity.TokenTypeAPIKey, token.TokenType)
	assert.Equal(t, "My API Key", token.Name)
	assert.Equal(t, expiresAt, token.ExpiresAt)
	assert.Equal(t, 60, token.RateLimitPerMinute)
	assert.Equal(t, 10000, token.RateLimitPerDay)
	assert.False(t, token.IsRevoked)
}

func TestToken_IsExpired(t *testing.T) {
	token := entity.NewAPIKey(uuid.New(), "hash", "key", time.Now().Add(1*time.Hour))
	assert.False(t, token.IsExpired())

	expiredToken := entity.NewAPIKey(uuid.New(), "hash", "key", time.Now().Add(-1*time.Hour))
	assert.True(t, expiredToken.IsExpired())
}

func TestToken_IsValid(t *testing.T) {
	token := entity.NewAPIKey(uuid.New(), "hash", "key", time.Now().Add(1*time.Hour))
	assert.True(t, token.IsValid())

	token.IsRevoked = true
	assert.False(t, token.IsValid())

	revokedExpired := entity.NewAPIKey(uuid.New(), "hash", "key", time.Now().Add(-1*time.Hour))
	assert.False(t, revokedExpired.IsValid())
}

func TestToken_CanUseTool(t *testing.T) {
	token := entity.NewAPIKey(uuid.New(), "hash", "key", time.Now().Add(1*time.Hour))

	// nil = all tools allowed
	assert.True(t, token.CanUseTool("web_search"))
	assert.True(t, token.CanUseTool("calculator"))

	// specific tools
	token.AllowedTools = []string{"calculator"}
	assert.True(t, token.CanUseTool("calculator"))
	assert.False(t, token.CanUseTool("web_search"))

	// wildcard
	token.AllowedTools = []string{"*"}
	assert.True(t, token.CanUseTool("anything"))
}

func TestToken_CanUseModel(t *testing.T) {
	token := entity.NewAPIKey(uuid.New(), "hash", "key", time.Now().Add(1*time.Hour))

	// nil = all models allowed
	assert.True(t, token.CanUseModel("gpt-4o"))

	// specific models
	token.AllowedModels = []string{"gpt-4o-mini"}
	assert.True(t, token.CanUseModel("gpt-4o-mini"))
	assert.False(t, token.CanUseModel("gpt-4o"))
}

func TestToken_UpdateLastUsed(t *testing.T) {
	token := entity.NewAPIKey(uuid.New(), "hash", "key", time.Now().Add(1*time.Hour))
	assert.Nil(t, token.LastUsedAt)

	token.UpdateLastUsed()
	assert.NotNil(t, token.LastUsedAt)
}
