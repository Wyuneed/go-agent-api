package service

import (
	"fmt"

	"github.com/wyuneed/go-agent-api/internal/domain/entity"
)

// AuthDomainService encapsulates authentication business rules
// that span across User and Token entities.
type AuthDomainService struct{}

func NewAuthDomainService() *AuthDomainService {
	return &AuthDomainService{}
}

// ValidateTokenAccess checks if a token grants access for a given user, tool, and model.
func (s *AuthDomainService) ValidateTokenAccess(user *entity.User, token *entity.Token, toolName, modelName string) error {
	if !user.IsActive {
		return fmt.Errorf("user account is inactive")
	}

	if !token.IsValid() {
		if token.IsExpired() {
			return fmt.Errorf("token has expired")
		}
		return fmt.Errorf("token has been revoked")
	}

	if token.UserID != user.ID {
		return fmt.Errorf("token does not belong to user")
	}

	if toolName != "" && !token.CanUseTool(toolName) {
		return fmt.Errorf("token does not have access to tool: %s", toolName)
	}

	if modelName != "" && !token.CanUseModel(modelName) {
		return fmt.Errorf("token does not have access to model: %s", modelName)
	}

	return nil
}

// CanCreateAPIKey checks if a user is allowed to create new API keys.
func (s *AuthDomainService) CanCreateAPIKey(user *entity.User) bool {
	return user.IsActive && (user.Role == entity.UserRoleAdmin || user.Role == entity.UserRoleUser)
}
