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

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

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
