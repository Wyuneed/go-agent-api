package handler

import (
	"encoding/json"
	"net/http"

	"github.com/wyuneed/go-agent-api/internal/application/dto/request"
	"github.com/wyuneed/go-agent-api/internal/application/usecase/auth"
	"github.com/wyuneed/go-agent-api/internal/application/usecase/user"
	"github.com/wyuneed/go-agent-api/internal/pkg/response"
)

type AuthHandler struct {
	login      *auth.LoginUseCase
	refresh    *auth.RefreshTokenUseCase
	createUser *user.CreateUserUseCase
}

func NewAuthHandler(login *auth.LoginUseCase, refresh *auth.RefreshTokenUseCase, createUser *user.CreateUserUseCase) *AuthHandler {
	return &AuthHandler{
		login:      login,
		refresh:    refresh,
		createUser: createUser,
	}
}

// Register godoc
// @Summary      Register a new user
// @Description  Creates a new user account with email and password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      request.RegisterRequest          true  "Registration details"
// @Success      201   {object}  response.SwaggerRegisterResponse
// @Failure      400   {object}  response.SwaggerErrorResponse
// @Router       /v1/auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req request.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	result, err := h.createUser.Execute(r.Context(), user.CreateUserInput{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		response.Error(w, http.StatusBadRequest, "REGISTER_ERROR", err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, result)
}

// Login godoc
// @Summary      Login
// @Description  Authenticates a user and returns JWT access + refresh tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      request.LoginRequest          true  "Login credentials"
// @Success      200   {object}  response.SwaggerLoginResponse
// @Failure      400   {object}  response.SwaggerErrorResponse
// @Failure      401   {object}  response.SwaggerErrorResponse
// @Router       /v1/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req request.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	result, err := h.login.Execute(r.Context(), auth.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "LOGIN_ERROR", err.Error())
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// RefreshToken godoc
// @Summary      Refresh access token
// @Description  Uses a valid refresh token to issue a new access token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      request.RefreshTokenRequest   true  "Refresh token"
// @Success      200   {object}  response.SwaggerLoginResponse
// @Failure      400   {object}  response.SwaggerErrorResponse
// @Failure      401   {object}  response.SwaggerErrorResponse
// @Router       /v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req request.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	result, err := h.refresh.Execute(r.Context(), auth.RefreshTokenInput{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "REFRESH_ERROR", err.Error())
		return
	}

	response.JSON(w, http.StatusOK, result)
}
