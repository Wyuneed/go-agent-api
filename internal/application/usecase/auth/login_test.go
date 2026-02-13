package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/wyuneed/go-agent-api/internal/application/usecase/auth"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
	jwtpkg "github.com/wyuneed/go-agent-api/internal/pkg/jwt"
	"github.com/wyuneed/go-agent-api/tests/mocks"
)

func newJWT() *jwtpkg.JWTManager {
	return jwtpkg.NewJWTManager("test-secret-32-chars-minimum!!", 15*time.Minute, 7*24*time.Hour)
}

func hashedPassword(t *testing.T, plain string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.MinCost)
	require.NoError(t, err)
	return string(h)
}

func TestLogin_Success(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockTokenRepository{}
	jwtMgr := newJWT()

	user := entity.NewUser("alice@example.com", hashedPassword(t, "Secret123"), "Alice")

	userRepo.On("FindByEmail", mock.Anything, "alice@example.com").Return(user, nil)
	tokenRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Token")).Return(nil)

	uc := auth.NewLoginUseCase(userRepo, tokenRepo, jwtMgr)
	out, err := uc.Execute(context.Background(), auth.LoginInput{
		Email:    "alice@example.com",
		Password: "Secret123",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, out.AccessToken)
	assert.NotEmpty(t, out.RefreshToken)
	assert.Equal(t, "Bearer", out.TokenType)
	assert.Equal(t, 900, out.ExpiresIn)
	assert.Equal(t, user.ID, out.User.ID)

	userRepo.AssertExpectations(t)
	tokenRepo.AssertExpectations(t)
}

func TestLogin_UserNotFound(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockTokenRepository{}

	userRepo.On("FindByEmail", mock.Anything, "nobody@example.com").Return(nil, errors.New("not found"))

	uc := auth.NewLoginUseCase(userRepo, tokenRepo, newJWT())
	_, err := uc.Execute(context.Background(), auth.LoginInput{
		Email:    "nobody@example.com",
		Password: "any",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
	tokenRepo.AssertNotCalled(t, "Create")
}

func TestLogin_WrongPassword(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockTokenRepository{}

	user := entity.NewUser("bob@example.com", hashedPassword(t, "CorrectPass"), "Bob")
	userRepo.On("FindByEmail", mock.Anything, "bob@example.com").Return(user, nil)

	uc := auth.NewLoginUseCase(userRepo, tokenRepo, newJWT())
	_, err := uc.Execute(context.Background(), auth.LoginInput{
		Email:    "bob@example.com",
		Password: "WrongPass",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
	tokenRepo.AssertNotCalled(t, "Create")
}

func TestLogin_InactiveUser(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockTokenRepository{}

	user := entity.NewUser("inactive@example.com", hashedPassword(t, "Pass123"), "Inactive")
	user.IsActive = false
	userRepo.On("FindByEmail", mock.Anything, "inactive@example.com").Return(user, nil)

	uc := auth.NewLoginUseCase(userRepo, tokenRepo, newJWT())
	_, err := uc.Execute(context.Background(), auth.LoginInput{
		Email:    "inactive@example.com",
		Password: "Pass123",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inactive")
	tokenRepo.AssertNotCalled(t, "Create")
}

func TestLogin_TokenCreateFails(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockTokenRepository{}

	user := entity.NewUser("eve@example.com", hashedPassword(t, "Pass123"), "Eve")
	userRepo.On("FindByEmail", mock.Anything, "eve@example.com").Return(user, nil)
	tokenRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Token")).Return(errors.New("db error"))

	uc := auth.NewLoginUseCase(userRepo, tokenRepo, newJWT())
	_, err := uc.Execute(context.Background(), auth.LoginInput{
		Email:    "eve@example.com",
		Password: "Pass123",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create token")
}

// ---- ValidateToken tests ----

func TestValidateToken_JWT_Success(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockTokenRepository{}
	jwtMgr := newJWT()

	user := entity.NewUser("val@example.com", "hash", "Val")
	tokenEntity := entity.NewAPIKey(user.ID, "hash", "session", time.Now().Add(time.Hour))
	tokenEntity.TokenType = entity.TokenTypeAccess

	jwtStr, err := jwtMgr.GenerateAccessToken(user.ID, tokenEntity.ID)
	require.NoError(t, err)

	tokenRepo.On("FindByID", mock.Anything, tokenEntity.ID).Return(tokenEntity, nil)
	userRepo.On("FindByID", mock.Anything, user.ID).Return(user, nil)
	// UpdateLastUsed is called in a goroutine — use Maybe so it's optional
	tokenRepo.On("UpdateLastUsed", mock.Anything, tokenEntity.ID).Maybe().Return(nil)

	uc := auth.NewValidateTokenUseCase(tokenRepo, userRepo, jwtMgr)
	result, err := uc.Execute(context.Background(), jwtStr)

	require.NoError(t, err)
	assert.Equal(t, user.ID, result.User.ID)
	assert.Equal(t, tokenEntity.ID, result.Token.ID)
}

func TestValidateToken_JWT_Invalid(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockTokenRepository{}

	uc := auth.NewValidateTokenUseCase(tokenRepo, userRepo, newJWT())
	_, err := uc.Execute(context.Background(), "not.a.jwt")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
}

func TestValidateToken_JWT_RevokedToken(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockTokenRepository{}
	jwtMgr := newJWT()

	userID := uuid.New()
	tokenEntity := entity.NewAPIKey(userID, "hash", "session", time.Now().Add(time.Hour))
	tokenEntity.IsRevoked = true

	jwtStr, err := jwtMgr.GenerateAccessToken(userID, tokenEntity.ID)
	require.NoError(t, err)

	tokenRepo.On("FindByID", mock.Anything, tokenEntity.ID).Return(tokenEntity, nil)

	uc := auth.NewValidateTokenUseCase(tokenRepo, userRepo, jwtMgr)
	_, err = uc.Execute(context.Background(), jwtStr)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired or revoked")
}

func TestValidateToken_JWT_InactiveUser(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockTokenRepository{}
	jwtMgr := newJWT()

	user := entity.NewUser("gone@example.com", "hash", "Gone")
	user.IsActive = false
	tokenEntity := entity.NewAPIKey(user.ID, "hash", "session", time.Now().Add(time.Hour))

	jwtStr, err := jwtMgr.GenerateAccessToken(user.ID, tokenEntity.ID)
	require.NoError(t, err)

	tokenRepo.On("FindByID", mock.Anything, tokenEntity.ID).Return(tokenEntity, nil)
	userRepo.On("FindByID", mock.Anything, user.ID).Return(user, nil)
	tokenRepo.On("UpdateLastUsed", mock.Anything, tokenEntity.ID).Maybe().Return(nil)

	uc := auth.NewValidateTokenUseCase(tokenRepo, userRepo, jwtMgr)
	_, err = uc.Execute(context.Background(), jwtStr)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inactive")
}

func TestValidateToken_APIKey_Success(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockTokenRepository{}
	jwtMgr := newJWT()

	rawKey := jwtpkg.GenerateAPIKey() // "sk-<uuid>"
	keyHash := jwtpkg.HashToken(rawKey)

	user := entity.NewUser("apiuser@example.com", "hash", "APIUser")
	tokenEntity := entity.NewAPIKey(user.ID, keyHash, "my-key", time.Now().Add(24*time.Hour))

	tokenRepo.On("FindByHash", mock.Anything, keyHash).Return(tokenEntity, nil)
	userRepo.On("FindByID", mock.Anything, user.ID).Return(user, nil)
	tokenRepo.On("UpdateLastUsed", mock.Anything, tokenEntity.ID).Maybe().Return(nil)

	uc := auth.NewValidateTokenUseCase(tokenRepo, userRepo, jwtMgr)
	result, err := uc.Execute(context.Background(), rawKey)

	require.NoError(t, err)
	assert.Equal(t, user.ID, result.User.ID)
}

func TestValidateToken_APIKey_NotFound(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockTokenRepository{}

	rawKey := jwtpkg.GenerateAPIKey()
	keyHash := jwtpkg.HashToken(rawKey)

	tokenRepo.On("FindByHash", mock.Anything, keyHash).Return(nil, errors.New("not found"))

	uc := auth.NewValidateTokenUseCase(tokenRepo, userRepo, newJWT())
	_, err := uc.Execute(context.Background(), rawKey)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid API key")
}

// ---- RefreshToken tests ----

func TestRefreshToken_Success(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockTokenRepository{}
	jwtMgr := newJWT()

	user := entity.NewUser("refresh@example.com", "hash", "Refresh")
	tokenEntity := entity.NewAPIKey(user.ID, "hash", "session", time.Now().Add(time.Hour))

	refreshStr, err := jwtMgr.GenerateRefreshToken(user.ID, tokenEntity.ID)
	require.NoError(t, err)

	tokenRepo.On("FindByID", mock.Anything, tokenEntity.ID).Return(tokenEntity, nil)
	userRepo.On("FindByID", mock.Anything, user.ID).Return(user, nil)

	uc := auth.NewRefreshTokenUseCase(tokenRepo, userRepo, jwtMgr)
	out, err := uc.Execute(context.Background(), auth.RefreshTokenInput{RefreshToken: refreshStr})

	require.NoError(t, err)
	assert.NotEmpty(t, out.AccessToken)
	assert.Equal(t, "Bearer", out.TokenType)
	assert.Equal(t, 900, out.ExpiresIn)
}

func TestRefreshToken_WrongTokenType(t *testing.T) {
	tokenRepo := &mocks.MockTokenRepository{}
	userRepo := &mocks.MockUserRepository{}
	jwtMgr := newJWT()

	// Generate an *access* token, not a refresh token
	accessStr, err := jwtMgr.GenerateAccessToken(uuid.New(), uuid.New())
	require.NoError(t, err)

	uc := auth.NewRefreshTokenUseCase(tokenRepo, userRepo, jwtMgr)
	_, err = uc.Execute(context.Background(), auth.RefreshTokenInput{RefreshToken: accessStr})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a refresh token")
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	uc := auth.NewRefreshTokenUseCase(
		&mocks.MockTokenRepository{},
		&mocks.MockUserRepository{},
		newJWT(),
	)
	_, err := uc.Execute(context.Background(), auth.RefreshTokenInput{RefreshToken: "garbage"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid refresh token")
}

func TestRefreshToken_RevokedToken(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockTokenRepository{}
	jwtMgr := newJWT()

	userID := uuid.New()
	tokenEntity := entity.NewAPIKey(userID, "hash", "session", time.Now().Add(time.Hour))
	tokenEntity.IsRevoked = true

	refreshStr, err := jwtMgr.GenerateRefreshToken(userID, tokenEntity.ID)
	require.NoError(t, err)

	tokenRepo.On("FindByID", mock.Anything, tokenEntity.ID).Return(tokenEntity, nil)

	uc := auth.NewRefreshTokenUseCase(tokenRepo, userRepo, jwtMgr)
	_, err = uc.Execute(context.Background(), auth.RefreshTokenInput{RefreshToken: refreshStr})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired or revoked")
}

func TestRefreshToken_InactiveUser(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockTokenRepository{}
	jwtMgr := newJWT()

	user := entity.NewUser("gone@example.com", "hash", "Gone")
	user.IsActive = false
	tokenEntity := entity.NewAPIKey(user.ID, "hash", "session", time.Now().Add(time.Hour))

	refreshStr, err := jwtMgr.GenerateRefreshToken(user.ID, tokenEntity.ID)
	require.NoError(t, err)

	tokenRepo.On("FindByID", mock.Anything, tokenEntity.ID).Return(tokenEntity, nil)
	userRepo.On("FindByID", mock.Anything, user.ID).Return(user, nil)

	uc := auth.NewRefreshTokenUseCase(tokenRepo, userRepo, jwtMgr)
	_, err = uc.Execute(context.Background(), auth.RefreshTokenInput{RefreshToken: refreshStr})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inactive")
}
