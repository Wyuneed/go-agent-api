//go:build integration

// Package integration contains integration tests that test multiple layers together.
// Run with: go test -v -tags=integration ./tests/integration/...
package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wyuneed/go-agent-api/internal/application/usecase/auth"
	"github.com/wyuneed/go-agent-api/internal/application/usecase/user"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
	"github.com/wyuneed/go-agent-api/internal/infrastructure/http/handler"
	"github.com/wyuneed/go-agent-api/internal/pkg/jwt"
	"github.com/wyuneed/go-agent-api/tests/mocks"
)

func buildAuthRouter(t *testing.T, userRepo *mocks.MockUserRepository, tokenRepo *mocks.MockTokenRepository) http.Handler {
	t.Helper()

	jwtManager := jwt.NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	createUserUC := user.NewCreateUserUseCase(userRepo)
	loginUC := auth.NewLoginUseCase(userRepo, tokenRepo, jwtManager)
	refreshUC := auth.NewRefreshTokenUseCase(tokenRepo, userRepo, jwtManager)

	authHandler := handler.NewAuthHandler(loginUC, refreshUC, createUserUC)

	r := chi.NewRouter()
	r.Post("/v1/auth/register", authHandler.Register)
	r.Post("/v1/auth/login", authHandler.Login)
	r.Post("/v1/auth/refresh", authHandler.RefreshToken)

	return r
}

func TestRegister_Success(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockTokenRepository{}

	userRepo.On("FindByEmail", mock.Anything, "new@example.com").Return(nil, nil)
	userRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.User")).Return(nil)

	router := buildAuthRouter(t, userRepo, tokenRepo)

	body := `{"email":"new@example.com","password":"SecurePass123!","name":"New User"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.True(t, resp["success"].(bool))

	userRepo.AssertExpectations(t)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockTokenRepository{}

	existing := entity.NewUser("existing@example.com", "hash", "Existing")
	userRepo.On("FindByEmail", mock.Anything, "existing@example.com").Return(existing, nil)

	router := buildAuthRouter(t, userRepo, tokenRepo)

	body := `{"email":"existing@example.com","password":"SecurePass123!","name":"Test"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should fail due to duplicate email (use case returns error)
	assert.NotEqual(t, http.StatusCreated, w.Code)
	userRepo.AssertExpectations(t)
}

func TestRegister_InvalidBody(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockTokenRepository{}

	router := buildAuthRouter(t, userRepo, tokenRepo)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockTokenRepository{}

	userRepo.On("FindByEmail", mock.Anything, "wrong@example.com").Return(nil, nil)

	router := buildAuthRouter(t, userRepo, tokenRepo)

	body := `{"email":"wrong@example.com","password":"wrongpass"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockTokenRepository{}

	router := buildAuthRouter(t, userRepo, tokenRepo)

	body := `{"refresh_token":"this-is-not-a-valid-jwt"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
