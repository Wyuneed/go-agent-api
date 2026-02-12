package jwt_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wyuneed/go-agent-api/internal/pkg/jwt"
)

func newTestManager() *jwt.JWTManager {
	return jwt.NewJWTManager("test-secret-key-for-testing", 15*time.Minute, 7*24*time.Hour)
}

func TestJWTManager_GenerateAndValidateAccessToken(t *testing.T) {
	manager := newTestManager()
	userID := uuid.New()
	tokenID := uuid.New()

	token, err := manager.GenerateAccessToken(userID, tokenID)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := manager.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, tokenID, claims.TokenID)
	assert.Equal(t, "access", claims.TokenType)
}

func TestJWTManager_GenerateAndValidateRefreshToken(t *testing.T) {
	manager := newTestManager()
	userID := uuid.New()
	tokenID := uuid.New()

	token, err := manager.GenerateRefreshToken(userID, tokenID)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := manager.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, "refresh", claims.TokenType)
}

func TestJWTManager_ValidateToken_InvalidToken(t *testing.T) {
	manager := newTestManager()

	_, err := manager.ValidateToken("this.is.not.a.valid.token")
	assert.Error(t, err)
}

func TestJWTManager_ValidateToken_WrongSecret(t *testing.T) {
	manager1 := jwt.NewJWTManager("secret-one", 15*time.Minute, 7*24*time.Hour)
	manager2 := jwt.NewJWTManager("secret-two", 15*time.Minute, 7*24*time.Hour)

	token, err := manager1.GenerateAccessToken(uuid.New(), uuid.New())
	require.NoError(t, err)

	_, err = manager2.ValidateToken(token)
	assert.Error(t, err)
}

func TestJWTManager_ValidateToken_Expired(t *testing.T) {
	// Create manager with very short TTL
	manager := jwt.NewJWTManager("secret", 1*time.Millisecond, 7*24*time.Hour)

	token, err := manager.GenerateAccessToken(uuid.New(), uuid.New())
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	_, err = manager.ValidateToken(token)
	assert.Error(t, err)
}

func TestHashToken(t *testing.T) {
	hash1 := jwt.HashToken("my-api-key")
	hash2 := jwt.HashToken("my-api-key")
	hash3 := jwt.HashToken("different-key")

	// Same input = same hash
	assert.Equal(t, hash1, hash2)
	// Different input = different hash
	assert.NotEqual(t, hash1, hash3)
	// Hash is non-empty hex string
	assert.NotEmpty(t, hash1)
	assert.Len(t, hash1, 64) // SHA-256 = 32 bytes = 64 hex chars
}

func TestGenerateAPIKey(t *testing.T) {
	key1 := jwt.GenerateAPIKey()
	key2 := jwt.GenerateAPIKey()

	assert.NotEmpty(t, key1)
	assert.NotEqual(t, key1, key2)
	assert.True(t, len(key1) > 3)
	assert.Equal(t, "sk-", key1[:3])
}
