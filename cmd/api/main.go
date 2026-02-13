// Package main is the entry point for the Go Agent API server.
//
// @title           Go Agent API
// @version         1.0
// @description     Production-ready AI Agent API with DDD architecture and Eino workflow engine. Supports OpenAI-compatible tool calling, human-in-the-loop approval, JWT authentication, and streaming responses.
// @termsOfService  http://swagger.io/terms/
//
// @contact.name   API Support
// @contact.url    https://github.com/wyuneed/go-agent-api/issues
//
// @license.name  MIT
// @license.url   https://github.com/wyuneed/go-agent-api/blob/main/LICENSE
//
// @host      127.0.0.1:8080
// @BasePath  /
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter "Bearer" followed by a space and your JWT token. Example: "Bearer eyJhbGci..."
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/wyuneed/go-agent-api/docs"
	"github.com/wyuneed/go-agent-api/internal/application/usecase/auth"
	"github.com/wyuneed/go-agent-api/internal/application/usecase/chat"
	"github.com/wyuneed/go-agent-api/internal/application/usecase/tool"
	"github.com/wyuneed/go-agent-api/internal/application/usecase/tool/builtin"
	"github.com/wyuneed/go-agent-api/internal/application/usecase/user"
	"github.com/wyuneed/go-agent-api/internal/infrastructure/config"
	"github.com/wyuneed/go-agent-api/internal/infrastructure/http/handler"
	"github.com/wyuneed/go-agent-api/internal/infrastructure/http/middleware"
	"github.com/wyuneed/go-agent-api/internal/infrastructure/llm/litellm"
	"github.com/wyuneed/go-agent-api/internal/infrastructure/persistence/postgres"
	"github.com/wyuneed/go-agent-api/internal/infrastructure/persistence/redis"
	"github.com/wyuneed/go-agent-api/internal/pkg/jwt"
)

func main() {
	// Load config
	cfg := config.Load()

	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Database
	db, err := postgres.NewConnection(ctx, cfg.Database)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Redis
	redisClient, err := redis.NewConnection(ctx, cfg.Redis)
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	// Repositories
	userRepo := postgres.NewUserRepository(db)
	tokenRepo := postgres.NewTokenRepository(db)
	convRepo := postgres.NewConversationRepository(db)
	msgRepo := postgres.NewMessageRepository(db)

	// JWT Manager
	jwtMgr := jwt.NewJWTManager(cfg.JWT.Secret, cfg.JWT.AccessTokenTTL, cfg.JWT.RefreshTokenTTL)

	// LLM Provider
	llmProvider := litellm.NewClient(cfg.LLM.BaseURL, cfg.LLM.APIKey)

	// Tool Registry
	toolRegistry := tool.NewToolRegistry()
	toolRegistry.RegisterAll(
		builtin.NewCalculatorTool(),
		builtin.NewWebSearchTool(cfg.Tools.WebSearchAPIKey),
	)

	// Use Cases
	validateTokenUC := auth.NewValidateTokenUseCase(tokenRepo, userRepo, jwtMgr)
	loginUC := auth.NewLoginUseCase(userRepo, tokenRepo, jwtMgr)
	refreshUC := auth.NewRefreshTokenUseCase(tokenRepo, userRepo, jwtMgr)
	createUserUC := user.NewCreateUserUseCase(userRepo)
	sendMessageUC := chat.NewSendMessageUseCase(convRepo, msgRepo, llmProvider)
	getConversationUC := chat.NewGetConversationUseCase(convRepo, msgRepo)
	listConversationsUC := chat.NewListConversationsUseCase(convRepo)
	approveActionUC := chat.NewApproveActionUseCase(convRepo)
	executeToolUC := tool.NewExecuteToolUseCase(toolRegistry, cfg.Tools.MaxConcurrent)

	// Middleware
	authMiddleware := middleware.NewAuthMiddleware(validateTokenUC)
	rateLimiter := middleware.NewRateLimiter(redisClient)

	// Handlers
	healthHandler := handler.NewHealthHandler(db, redisClient)
	authHandler := handler.NewAuthHandler(loginUC, refreshUC, createUserUC)
	chatHandler := handler.NewChatHandler(sendMessageUC, getConversationUC, listConversationsUC, approveActionUC)
	toolHandler := handler.NewToolHandler(toolRegistry, executeToolUC)

	// Router
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.Logger(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.CORS)

	// Swagger UI
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	// Health routes
	r.Get("/health", healthHandler.Health)
	r.Get("/ready", healthHandler.Ready)

	// API routes
	r.Route("/v1", func(r chi.Router) {
		// Public routes
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/refresh", authHandler.RefreshToken)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			if cfg.RateLimit.Enabled {
				r.Use(rateLimiter.Limit)
			}

			// Chat
			r.Post("/chat/completions", chatHandler.ChatCompletions)
			r.Post("/conversations", chatHandler.CreateConversation)
			r.Get("/conversations", chatHandler.ListConversations)
			r.Get("/conversations/{id}", chatHandler.GetConversation)
			r.Post("/conversations/{id}/messages", chatHandler.SendMessage)
			r.Post("/conversations/{id}/approve", chatHandler.ApproveAction)

			// Tools
			r.Get("/tools", toolHandler.ListTools)
			r.Post("/tools/execute", toolHandler.ExecuteTool)
			r.Post("/tools/batch", toolHandler.ExecuteToolBatch)
		})
	})

	// Server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server
	go func() {
		slog.Info("starting server", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}

	slog.Info("server stopped")
}
