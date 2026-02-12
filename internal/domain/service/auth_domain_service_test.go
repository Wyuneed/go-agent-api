package service_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
	"github.com/wyuneed/go-agent-api/internal/domain/service"
)

func TestAuthDomainService_ValidateTokenAccess(t *testing.T) {
	svc := service.NewAuthDomainService()
	userID := uuid.New()

	user := entity.NewUser("test@example.com", "hash", "Test User")
	user.ID = userID

	validToken := entity.NewAPIKey(userID, "hash", "key", time.Now().Add(1*time.Hour))

	t.Run("valid token and active user", func(t *testing.T) {
		err := svc.ValidateTokenAccess(user, validToken, "", "")
		assert.NoError(t, err)
	})

	t.Run("inactive user", func(t *testing.T) {
		inactiveUser := *user
		inactiveUser.IsActive = false
		err := svc.ValidateTokenAccess(&inactiveUser, validToken, "", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "inactive")
	})

	t.Run("revoked token", func(t *testing.T) {
		revokedToken := *validToken
		revokedToken.IsRevoked = true
		err := svc.ValidateTokenAccess(user, &revokedToken, "", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "revoked")
	})

	t.Run("expired token", func(t *testing.T) {
		expiredToken := entity.NewAPIKey(userID, "hash", "key", time.Now().Add(-1*time.Hour))
		err := svc.ValidateTokenAccess(user, expiredToken, "", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("token user ID mismatch", func(t *testing.T) {
		mismatchToken := entity.NewAPIKey(uuid.New(), "hash", "key", time.Now().Add(1*time.Hour))
		err := svc.ValidateTokenAccess(user, mismatchToken, "", "")
		assert.Error(t, err)
	})

	t.Run("tool not allowed", func(t *testing.T) {
		restrictedToken := *validToken
		restrictedToken.AllowedTools = []string{"calculator"}
		err := svc.ValidateTokenAccess(user, &restrictedToken, "web_search", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "web_search")
	})

	t.Run("model not allowed", func(t *testing.T) {
		restrictedToken := *validToken
		restrictedToken.AllowedModels = []string{"gpt-4o-mini"}
		err := svc.ValidateTokenAccess(user, &restrictedToken, "", "gpt-4o")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "gpt-4o")
	})
}

func TestAuthDomainService_CanCreateAPIKey(t *testing.T) {
	svc := service.NewAuthDomainService()

	t.Run("admin can create", func(t *testing.T) {
		user := entity.NewUser("admin@example.com", "hash", "Admin")
		user.Role = entity.UserRoleAdmin
		assert.True(t, svc.CanCreateAPIKey(user))
	})

	t.Run("user can create", func(t *testing.T) {
		user := entity.NewUser("user@example.com", "hash", "User")
		assert.True(t, svc.CanCreateAPIKey(user))
	})

	t.Run("inactive user cannot create", func(t *testing.T) {
		user := entity.NewUser("inactive@example.com", "hash", "Inactive")
		user.IsActive = false
		assert.False(t, svc.CanCreateAPIKey(user))
	})
}
