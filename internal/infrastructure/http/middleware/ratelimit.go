package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wyuneed/go-agent-api/internal/pkg/response"
)

type RateLimiter struct {
	redis *redis.Client
}

func NewRateLimiter(redis *redis.Client) *RateLimiter {
	return &RateLimiter{redis: redis}
}

func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := GetTokenFromContext(r.Context())
		if token == nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()

		// Check per-minute limit
		minuteKey := fmt.Sprintf("ratelimit:%s:minute:%d", token.ID, time.Now().Unix()/60)
		count, err := rl.redis.Incr(ctx, minuteKey).Result()
		if err == nil && count == 1 {
			rl.redis.Expire(ctx, minuteKey, time.Minute)
		}

		if count > int64(token.RateLimitPerMinute) {
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", token.RateLimitPerMinute))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("Retry-After", "60")
			response.Error(w, http.StatusTooManyRequests, "RATE_LIMIT", "Rate limit exceeded")
			return
		}

		// Check per-day limit
		dayKey := fmt.Sprintf("ratelimit:%s:day:%s", token.ID, time.Now().Format("2006-01-02"))
		dayCount, err := rl.redis.Incr(ctx, dayKey).Result()
		if err == nil && dayCount == 1 {
			rl.redis.Expire(ctx, dayKey, 24*time.Hour)
		}

		if dayCount > int64(token.RateLimitPerDay) {
			response.Error(w, http.StatusTooManyRequests, "RATE_LIMIT", "Daily rate limit exceeded")
			return
		}

		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", token.RateLimitPerMinute))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", token.RateLimitPerMinute-int(count)))

		next.ServeHTTP(w, r)
	})
}
