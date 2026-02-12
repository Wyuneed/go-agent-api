package entity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
)

func TestNewUser(t *testing.T) {
	user := entity.NewUser("test@example.com", "hashed_password", "Test User")

	assert.NotNil(t, user)
	assert.NotEqual(t, user.ID.String(), "00000000-0000-0000-0000-000000000000")
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "hashed_password", user.PasswordHash)
	assert.Equal(t, "Test User", user.Name)
	assert.Equal(t, entity.UserRoleUser, user.Role)
	assert.True(t, user.IsActive)
	assert.NotNil(t, user.Metadata)
	assert.False(t, user.CreatedAt.IsZero())
	assert.False(t, user.UpdatedAt.IsZero())
}

func TestUser_CanUseTools(t *testing.T) {
	tests := []struct {
		name     string
		role     entity.UserRole
		isActive bool
		want     bool
	}{
		{"active admin", entity.UserRoleAdmin, true, true},
		{"active user", entity.UserRoleUser, true, true},
		{"active agent", entity.UserRoleAgent, true, false},
		{"inactive admin", entity.UserRoleAdmin, false, false},
		{"inactive user", entity.UserRoleUser, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := entity.NewUser("test@example.com", "hash", "Name")
			user.Role = tt.role
			user.IsActive = tt.isActive
			assert.Equal(t, tt.want, user.CanUseTools())
		})
	}
}

func TestUser_IsAdmin(t *testing.T) {
	user := entity.NewUser("test@example.com", "hash", "Name")
	assert.False(t, user.IsAdmin())

	user.Role = entity.UserRoleAdmin
	assert.True(t, user.IsAdmin())
}
