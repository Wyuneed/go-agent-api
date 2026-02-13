package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/wyuneed/go-agent-api/internal/domain/entity"
	"github.com/wyuneed/go-agent-api/internal/domain/repository"
	"github.com/wyuneed/go-agent-api/internal/pkg/jwt"
	"golang.org/x/crypto/bcrypt"
)

type LoginInput struct {
	Email    string
	Password string
}

type LoginOutput struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int          `json:"expires_in"`
	User         *entity.User `json:"user"`
}

type LoginUseCase struct {
	userRepo  repository.UserRepository
	tokenRepo repository.TokenRepository
	jwtMgr    *jwt.JWTManager
}

func NewLoginUseCase(userRepo repository.UserRepository, tokenRepo repository.TokenRepository, jwtMgr *jwt.JWTManager) *LoginUseCase {
	return &LoginUseCase{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		jwtMgr:    jwtMgr,
	}
}

func (uc *LoginUseCase) Execute(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	user, err := uc.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if !user.IsActive {
		return nil, fmt.Errorf("user account is inactive")
	}

	// Create token record; use a random nonce as the hash (JWT sessions are looked
	// up by token ID from JWT claims, not by hash, so it just needs to be unique).
	tokenEntity := entity.NewAPIKey(user.ID, jwt.HashToken(jwt.GenerateAPIKey()), "session", time.Now().Add(7*24*time.Hour))
	tokenEntity.TokenType = entity.TokenTypeAccess

	if err := uc.tokenRepo.Create(ctx, tokenEntity); err != nil {
		return nil, fmt.Errorf("create token: %w", err)
	}

	accessToken, err := uc.jwtMgr.GenerateAccessToken(user.ID, tokenEntity.ID)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := uc.jwtMgr.GenerateRefreshToken(user.ID, tokenEntity.ID)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	return &LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    900, // 15 minutes
		User:         user,
	}, nil
}
