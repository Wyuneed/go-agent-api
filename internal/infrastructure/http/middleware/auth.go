package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/wyuneed/go-agent-api/internal/application/usecase/auth"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
	"github.com/wyuneed/go-agent-api/internal/pkg/response"
)

type contextKey string

const (
	UserContextKey  contextKey = "user"
	TokenContextKey contextKey = "token"
)

type AuthMiddleware struct {
	validateToken *auth.ValidateTokenUseCase
}

func NewAuthMiddleware(validateToken *auth.ValidateTokenUseCase) *AuthMiddleware {
	return &AuthMiddleware{validateToken: validateToken}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authorization header")
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		token = strings.TrimSpace(token)

		result, err := m.validateToken.Execute(r.Context(), token)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", err.Error())
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, result.User)
		ctx = context.WithValue(ctx, TokenContextKey, result.Token)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		token = strings.TrimSpace(token)

		result, err := m.validateToken.Execute(r.Context(), token)
		if err == nil {
			ctx := context.WithValue(r.Context(), UserContextKey, result.User)
			ctx = context.WithValue(ctx, TokenContextKey, result.Token)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}

func GetUserFromContext(ctx context.Context) *entity.User {
	user, ok := ctx.Value(UserContextKey).(*entity.User)
	if !ok {
		return nil
	}
	return user
}

func GetTokenFromContext(ctx context.Context) *entity.Token {
	token, ok := ctx.Value(TokenContextKey).(*entity.Token)
	if !ok {
		return nil
	}
	return token
}
