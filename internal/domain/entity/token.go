package entity

import (
	"time"

	"github.com/google/uuid"
)

type TokenType string

const (
	TokenTypeAPIKey  TokenType = "api_key"
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

type Token struct {
	ID                 uuid.UUID
	UserID             uuid.UUID
	TokenHash          string
	TokenType          TokenType
	Name               string
	ExpiresAt          time.Time
	LastUsedAt         *time.Time
	RateLimitPerMinute int
	RateLimitPerDay    int
	AllowedTools       []string
	AllowedModels      []string
	IsRevoked          bool
	Metadata           map[string]any
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func NewAPIKey(userID uuid.UUID, tokenHash, name string, expiresAt time.Time) *Token {
	return &Token{
		ID:                 uuid.New(),
		UserID:             userID,
		TokenHash:          tokenHash,
		TokenType:          TokenTypeAPIKey,
		Name:               name,
		ExpiresAt:          expiresAt,
		RateLimitPerMinute: 60,
		RateLimitPerDay:    10000,
		IsRevoked:          false,
		Metadata:           make(map[string]any),
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
}

func (t *Token) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

func (t *Token) IsValid() bool {
	return !t.IsRevoked && !t.IsExpired()
}

func (t *Token) CanUseTool(toolName string) bool {
	if len(t.AllowedTools) == 0 {
		return true
	}
	for _, allowed := range t.AllowedTools {
		if allowed == toolName || allowed == "*" {
			return true
		}
	}
	return false
}

func (t *Token) CanUseModel(modelName string) bool {
	if len(t.AllowedModels) == 0 {
		return true
	}
	for _, allowed := range t.AllowedModels {
		if allowed == modelName || allowed == "*" {
			return true
		}
	}
	return false
}

func (t *Token) UpdateLastUsed() {
	now := time.Now()
	t.LastUsedAt = &now
}
