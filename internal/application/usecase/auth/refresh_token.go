package auth

import (
	"context"
	"fmt"

	"github.com/wyuneed/go-agent-api/internal/domain/repository"
	"github.com/wyuneed/go-agent-api/internal/pkg/jwt"
)

type RefreshTokenInput struct {
	RefreshToken string
}

type RefreshTokenOutput struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type RefreshTokenUseCase struct {
	tokenRepo repository.TokenRepository
	userRepo  repository.UserRepository
	jwtMgr    *jwt.JWTManager
}

func NewRefreshTokenUseCase(tokenRepo repository.TokenRepository, userRepo repository.UserRepository, jwtMgr *jwt.JWTManager) *RefreshTokenUseCase {
	return &RefreshTokenUseCase{
		tokenRepo: tokenRepo,
		userRepo:  userRepo,
		jwtMgr:    jwtMgr,
	}
}

func (uc *RefreshTokenUseCase) Execute(ctx context.Context, input RefreshTokenInput) (*RefreshTokenOutput, error) {
	claims, err := uc.jwtMgr.ValidateToken(input.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	if claims.TokenType != "refresh" {
		return nil, fmt.Errorf("not a refresh token")
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

	accessToken, err := uc.jwtMgr.GenerateAccessToken(user.ID, token.ID)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	return &RefreshTokenOutput{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   900,
	}, nil
}
