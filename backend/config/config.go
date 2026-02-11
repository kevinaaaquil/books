package config

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port                      string
	MongoURI                  string
	DBName                    string
	S3Bucket                  string
	S3Region                  string
	S3AccessKeyID             string
	S3SecretKey               string
	AuthEmail                 string
	AuthPass                  string
	JWTSecret                 string
	MaxUploadMB               int64
	EmailConfigEncryptionKey  []byte // 32 bytes for AES-256; optional, base64 in env
}

func Load() (*Config, error) {
	_ = os.Setenv("AWS_REGION", getEnv("AWS_REGION", "us-east-1"))
	maxMB := int64(50)
	if v := getEnv("MAX_UPLOAD_MB", "50"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			maxMB = n
		}
	}
	var emailEncKey []byte
	if k := getEnv("KINDLE_CONFIG_ENCRYPTION_KEY", ""); k != "" {
		emailEncKey, _ = base64.StdEncoding.DecodeString(k)
		if len(emailEncKey) != 32 {
			emailEncKey = nil
		}
	}

	return &Config{
		Port:                     getEnv("PORT", "8080"),
		MongoURI:                 getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		DBName:                   getEnv("MONGODB_DB", "books"),
		S3Bucket:                 getEnv("AWS_S3_BUCKET", ""),
		S3Region:                 getEnv("AWS_REGION", "us-east-1"),
		S3AccessKeyID:            getEnv("AWS_ACCESS_KEY_ID", ""),
		S3SecretKey:              getEnv("AWS_SECRET_ACCESS_KEY", ""),
		AuthEmail:                getEnv("AUTH_EMAIL", "user@example.com"),
		AuthPass:                 getEnv("AUTH_PASSWORD", "password"),
		JWTSecret:                getEnv("JWT_SECRET", "change-me-in-production"),
		MaxUploadMB:              maxMB,
		EmailConfigEncryptionKey: emailEncKey,
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// RequiredEnvVars are checked at startup; app exits if any are unset.
var RequiredEnvVars = []string{
	"MONGODB_URI",
	"MONGODB_DB",
	"JWT_SECRET",
	"AUTH_EMAIL",
	"AUTH_PASSWORD",
	"AWS_S3_BUCKET",
	"AWS_REGION",
	"AWS_ACCESS_KEY_ID",
	"AWS_SECRET_ACCESS_KEY",
	"MAX_UPLOAD_MB",
	"KINDLE_CONFIG_ENCRYPTION_KEY",
}

// OptionalEnvVars are logged at startup so you can confirm they are loaded when set.
var OptionalEnvVars = []string{
	"PORT",
}

// ValidateEnv checks that all required env vars are set and logs status of required + optional.
// Calls log.Fatal if any required var is missing.
func ValidateEnv() {
	var missing []string
	for _, key := range RequiredEnvVars {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			missing = append(missing, key)
		} else {
			log.Printf("env %s loaded", key)
		}
	}
	if len(missing) > 0 {
		log.Fatalf("missing required env: %s (set these in .env or environment)", strings.Join(missing, ", "))
	}
	for _, key := range OptionalEnvVars {
		v := strings.TrimSpace(os.Getenv(key))
		if v != "" {
			// Don't log secret values
			if key == "KINDLE_CONFIG_ENCRYPTION_KEY" || key == "AWS_ACCESS_KEY_ID" || key == "AWS_SECRET_ACCESS_KEY" || key == "AUTH_PASSWORD" {
				log.Printf("env %s loaded", key)
			} else {
				log.Printf("env %s = %s", key, v)
			}
		} else {
			log.Printf("env %s not set (optional)", key)
		}
	}
	if j := os.Getenv("JWT_SECRET"); j == "change-me-in-production" {
		log.Fatal("JWT_SECRET must be set to a strong secret (not the default change-me-in-production)")
	}
	k := os.Getenv("KINDLE_CONFIG_ENCRYPTION_KEY")
	if k == "" {
		log.Fatal("KINDLE_CONFIG_ENCRYPTION_KEY is required (generate with: openssl rand -base64 32)")
	}
	dec, _ := base64.StdEncoding.DecodeString(k)
	if len(dec) != 32 {
		log.Fatalf("KINDLE_CONFIG_ENCRYPTION_KEY must be 32 bytes base64 (got %d bytes); generate with: openssl rand -base64 32", len(dec))
	}
	fmt.Println("env check complete")
}
