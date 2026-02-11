package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/wyuneed/go-agent-api/internal/domain/entity"
	"github.com/wyuneed/go-agent-api/internal/domain/repository"
	"github.com/wyuneed/go-agent-api/internal/pkg/jwt"
)

type ValidateTokenResult struct {
	User  *entity.User
	Token *entity.Token
}

type ValidateTokenUseCase struct {
	tokenRepo repository.TokenRepository
	userRepo  repository.UserRepository
	jwtMgr    *jwt.JWTManager
}

func NewValidateTokenUseCase(tokenRepo repository.TokenRepository, userRepo repository.UserRepository, jwtMgr *jwt.JWTManager) *ValidateTokenUseCase {
	return &ValidateTokenUseCase{
		tokenRepo: tokenRepo,
		userRepo:  userRepo,
		jwtMgr:    jwtMgr,
	}
}

func (uc *ValidateTokenUseCase) Execute(ctx context.Context, rawToken string) (*ValidateTokenResult, error) {
	// Check if it's an API key (starts with "sk-")
	if strings.HasPrefix(rawToken, "sk-") {
		return uc.validateAPIKey(ctx, rawToken)
	}

	// Otherwise treat as JWT
	return uc.validateJWT(ctx, rawToken)
}

func (uc *ValidateTokenUseCase) validateAPIKey(ctx context.Context, rawKey string) (*ValidateTokenResult, error) {
	tokenHash := jwt.HashToken(rawKey)

	token, err := uc.tokenRepo.FindByHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("invalid API key")
	}

	if !token.IsValid() {
		return nil, fmt.Errorf("API key is expired or revoked")
	}

	user, err := uc.userRepo.FindByID(ctx, token.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	if !user.IsActive {
		return nil, fmt.Errorf("user account is inactive")
	}

	// Update last used (fire and forget)
	go uc.tokenRepo.UpdateLastUsed(context.Background(), token.ID)

	return &ValidateTokenResult{
		User:  user,
		Token: token,
	}, nil
}

func (uc *ValidateTokenUseCase) validateJWT(ctx context.Context, tokenString string) (*ValidateTokenResult, error) {
	claims, err := uc.jwtMgr.ValidateToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	token, err := uc.tokenRepo.FindByID(ctx, claims.TokenID)
	if err != nil {
		return nil, fmt.Errorf("token not found")
	}

	if !token.IsValid() {
		return nil, fmt.Errorf("token is expired or revoked")
	}

	user, err := uc.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	if !user.IsActive {
		return nil, fmt.Errorf("user account is inactive")
	}

	go uc.tokenRepo.UpdateLastUsed(context.Background(), token.ID)

	return &ValidateTokenResult{
		User:  user,
		Token: token,
	}, nil
}
