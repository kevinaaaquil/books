package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port            string
	MongoURI        string
	DBName          string
	S3Bucket        string
	S3Region        string
	S3AccessKeyID   string
	S3SecretKey     string
	AuthEmail       string
	AuthPass        string
	JWTSecret       string
	MaxUploadMB     int64
}

func Load() (*Config, error) {
	_ = os.Setenv("AWS_REGION", getEnv("AWS_REGION", "us-east-1"))
	maxMB := int64(50)
	if v := getEnv("MAX_UPLOAD_MB", "50"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			maxMB = n
		}
	}
	return &Config{
		Port:        getEnv("PORT", "8080"),
		MongoURI:    getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		DBName:      getEnv("MONGODB_DB", "books"),
		S3Bucket:      getEnv("AWS_S3_BUCKET", ""),
		S3Region:      getEnv("AWS_REGION", "us-east-1"),
		S3AccessKeyID: getEnv("AWS_ACCESS_KEY_ID", ""),
		S3SecretKey:   getEnv("AWS_SECRET_ACCESS_KEY", ""),
		AuthEmail:   getEnv("AUTH_EMAIL", "user@example.com"),
		AuthPass:    getEnv("AUTH_PASSWORD", "password"),
		JWTSecret:   getEnv("JWT_SECRET", "change-me-in-production"),
		MaxUploadMB: maxMB,
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
