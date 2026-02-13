package handler

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/wyuneed/go-agent-api/internal/pkg/response"
)

type HealthHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewHealthHandler(db *pgxpool.Pool, redis *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: redis}
}

// Health godoc
// @Summary      Liveness check
// @Description  Returns 200 OK when the server process is running
// @Tags         health
// @Produce      json
// @Success      200  {object}  response.SwaggerHealthResponse
// @Router       /health [get]
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Ready godoc
// @Summary      Readiness check
// @Description  Returns 200 OK when the server is ready (DB + Redis connected)
// @Tags         health
// @Produce      json
// @Success      200  {object}  response.SwaggerHealthResponse
// @Failure      503  {object}  response.SwaggerErrorResponse
// @Router       /ready [get]
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := h.db.Ping(ctx); err != nil {
		response.Error(w, http.StatusServiceUnavailable, "DB_ERROR", "Database not ready")
		return
	}

	if err := h.redis.Ping(ctx).Err(); err != nil {
		response.Error(w, http.StatusServiceUnavailable, "REDIS_ERROR", "Redis not ready")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
