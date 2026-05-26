package config

import (
	"os"
	"strings"
)

// Config загружается из переменных окружения (см. deploy/.env.example).
type Config struct {
	HTTPPort    string
	DatabaseURL string
	JWTSecret   string
	S3Endpoint  string
	S3AccessKey string
	S3SecretKey string
	S3Bucket    string
	S3UseSSL    bool
	IAMMock     bool
	AIMock      bool
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvBool(key string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if v == "" {
		return def
	}
	return v == "1" || v == "true" || v == "yes"
}

func Load() Config {
	return Config{
		HTTPPort:    getenv("HTTP_PORT", "8080"),
		DatabaseURL: getenv("DATABASE_URL", "postgres://itshop:itshop@localhost:5433/itshop?sslmode=disable"),
		JWTSecret:   getenv("JWT_SECRET", "dev-itshop-jwt-change-in-production"),
		S3Endpoint:  stripSchemeHostPort(getenv("S3_ENDPOINT", "http://localhost:9000")),
		S3AccessKey: getenv("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey: getenv("S3_SECRET_KEY", "minioadmin"),
		S3Bucket:    getenv("S3_BUCKET", "itshop-attachments"),
		S3UseSSL:    getenvBool("S3_USE_SSL", false),
		IAMMock:     getenvBool("IAM_MOCK", true),
		AIMock:      getenvBool("AI_MOCK", true),
	}
}

func stripSchemeHostPort(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "https://")
	raw = strings.TrimPrefix(raw, "http://")
	return raw
}
