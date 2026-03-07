package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Env      string
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Exchange ExchangeConfig
	AI       AIConfig
}

type ServerConfig struct {
	HTTPPort string
	GRPCPort string
}

type DatabaseConfig struct {
	URL string
}

type JWTConfig struct {
	Secret        string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

type ExchangeConfig struct {
	APIKey string
}

type AIConfig struct {
	Provider string
	APIKey   string
	Model    string
	BaseURL  string
}

func Load() *Config {
	return &Config{
		Env: getEnv("ENV", "development"),
		Server: ServerConfig{
			HTTPPort: getEnv("SERVER_HTTP_PORT", "8080"),
			GRPCPort: getEnv("SERVER_GRPC_PORT", "9090"),
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", "postgres://budget:budget@localhost:5432/budget?sslmode=disable"),
		},
		JWT: JWTConfig{
			Secret:        getEnv("JWT_SECRET", "your-secret-key-min-32-characters!!"),
			AccessExpiry:  parseDuration(getEnv("JWT_ACCESS_EXPIRY", "15m"), 15*time.Minute),
			RefreshExpiry: parseDuration(getEnv("JWT_REFRESH_EXPIRY", "30d"), 30*24*time.Hour), // 30 days
		},
		Exchange: ExchangeConfig{
			APIKey: getEnv("EXCHANGE_RATE_API_KEY", ""),
		},
		AI: AIConfig{
			Provider: getEnv("AI_PROVIDER", "openai"),
			APIKey:   getEnv("AI_API_KEY", ""),
			Model:    getEnv("AI_MODEL", "gpt-4o-mini"),
			BaseURL:  getEnv("AI_BASE_URL", "https://api.openai.com"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDuration(value string, defaultDuration time.Duration) time.Duration {
	// Handle "7d" format (days)
	if len(value) > 0 && value[len(value)-1] == 'd' {
		days, err := strconv.Atoi(value[:len(value)-1])
		if err == nil {
			return time.Duration(days) * 24 * time.Hour
		}
	}

	d, err := time.ParseDuration(value)
	if err != nil {
		return defaultDuration
	}
	return d
}
