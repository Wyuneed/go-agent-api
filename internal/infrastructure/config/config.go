package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	JWT       JWTConfig
	RateLimit RateLimitConfig
	LLM       LLMConfig
	Tools     ToolsConfig
}

type ServerConfig struct {
	Port            string
	Host            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}

type RedisConfig struct {
	Host         string
	Port         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
}

func (c RedisConfig) Addr() string {
	return c.Host + ":" + c.Port
}

type JWTConfig struct {
	Secret          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type RateLimitConfig struct {
	RequestsPerMinute int
	RequestsPerDay    int
	Enabled           bool
}

type LLMConfig struct {
	Provider     string
	BaseURL      string
	APIKey       string
	DefaultModel string
}

type ToolsConfig struct {
	WebSearchAPIKey string
	MaxConcurrent   int
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            getEnv("SERVER_PORT", "8080"),
			Host:            getEnv("SERVER_HOST", "0.0.0.0"),
			ReadTimeout:     getDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout:    getDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:     getDuration("SERVER_IDLE_TIMEOUT", 120*time.Second),
			ShutdownTimeout: getDuration("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			Name:            getEnv("DB_NAME", "aiagent"),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns:    getInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
			ConnMaxIdleTime: getDuration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Host:         getEnv("REDIS_HOST", "localhost"),
			Port:         getEnv("REDIS_PORT", "6379"),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           getInt("REDIS_DB", 0),
			PoolSize:     getInt("REDIS_POOL_SIZE", 10),
			MinIdleConns: getInt("REDIS_MIN_IDLE_CONNS", 5),
		},
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", "change-me-in-production-please"),
			AccessTokenTTL:  getDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTokenTTL: getDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
		},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: getInt("RATE_LIMIT_PER_MINUTE", 60),
			RequestsPerDay:    getInt("RATE_LIMIT_PER_DAY", 10000),
			Enabled:           getBool("RATE_LIMIT_ENABLED", true),
		},
		LLM: LLMConfig{
			Provider:     getEnv("LLM_PROVIDER", "litellm"),
			BaseURL:      getEnv("LLM_BASE_URL", "http://localhost:4000"),
			APIKey:       getEnv("LLM_API_KEY", ""),
			DefaultModel: getEnv("LLM_DEFAULT_MODEL", "gpt-4o-mini"),
		},
		Tools: ToolsConfig{
			WebSearchAPIKey: getEnv("WEB_SEARCH_API_KEY", ""),
			MaxConcurrent:   getInt("TOOLS_MAX_CONCURRENT", 10),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func getDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
