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
