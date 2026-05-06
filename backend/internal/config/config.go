package config

import (
	"fmt"
	"os"
	"strconv"
)

const insecureDefaultSecret = "dev-secret-key-change-in-production"

// Config holds all application configuration loaded from environment variables.
type Config struct {
	DBHost                string
	DBPort                string
	DBUser                string
	DBPassword            string
	DBName                string
	JWTSecret             string
	JWTExpiryHrs          int
	UploadDir             string
	Port                  string
	GoogleCredentialsFile string
	SMTPHost              string
	SMTPPort              string
	SMTPUser              string
	SMTPPassword          string
	SMTPFrom              string
	MaxLoginAttempts      int
	LockDurationMinutes   int
	PhotoRetentionDays    int
	AdminDefaultPassword  string
	LineChannelAccessToken string
}

// AppConfig is the global configuration instance.
var AppConfig *Config

// Load reads environment variables and populates the Config struct.
func Load() *Config {
	expiryHours := 24
	if v := os.Getenv("JWT_EXPIRY_HOURS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			expiryHours = parsed
		}
	}

	maxLoginAttempts := 5
	if v := os.Getenv("MAX_LOGIN_ATTEMPTS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			maxLoginAttempts = parsed
		}
	}

	lockDurationMinutes := 15
	if v := os.Getenv("LOCK_DURATION_MINUTES"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			lockDurationMinutes = parsed
		}
	}

	photoRetentionDays := 90
	if v := os.Getenv("PHOTO_RETENTION_DAYS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			photoRetentionDays = parsed
		}
	}

	cfg := &Config{
		DBHost:                getEnv("DB_HOST", "localhost"),
		DBPort:                getEnv("DB_PORT", "5432"),
		DBUser:                getEnv("DB_USER", "postgres"),
		DBPassword:            getEnv("DB_PASSWORD", "postgres"),
		DBName:                getEnv("DB_NAME", "translator_checkin"),
		JWTSecret:             getEnv("JWT_SECRET", "dev-secret-key-change-in-production"),
		JWTExpiryHrs:          expiryHours,
		UploadDir:             getEnv("UPLOAD_DIR", "./uploads"),
		Port:                  getEnv("PORT", "8080"),
		GoogleCredentialsFile: getEnv("GOOGLE_CREDENTIALS_FILE", ""),
		SMTPHost:              getEnv("SMTP_HOST", ""),
		SMTPPort:              getEnv("SMTP_PORT", "587"),
		SMTPUser:              getEnv("SMTP_USER", ""),
		SMTPPassword:          getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:               getEnv("SMTP_FROM", ""),
		MaxLoginAttempts:       maxLoginAttempts,
		LockDurationMinutes:   lockDurationMinutes,
		PhotoRetentionDays:    photoRetentionDays,
		AdminDefaultPassword:  getEnv("ADMIN_DEFAULT_PASSWORD", ""),
		LineChannelAccessToken: getEnv("LINE_CHANNEL_ACCESS_TOKEN", ""),
	}

	// Enforce strong JWT_SECRET — refuse to boot with the insecure default
	// or any secret shorter than 32 characters.
	if cfg.JWTSecret == insecureDefaultSecret || len(cfg.JWTSecret) < 32 {
		fmt.Fprintln(os.Stderr,
			"FATAL: JWT_SECRET is insecure (default value or length < 32 characters).\n"+
				"Set a strong secret in the JWT_SECRET environment variable before starting.\n"+
				"Generate one with: openssl rand -hex 32")
		os.Exit(1)
	}

	AppConfig = cfg
	return cfg
}

// DSN returns the PostgreSQL connection string.
func (c *Config) DSN() string {
	return "host=" + c.DBHost +
		" user=" + c.DBUser +
		" password=" + c.DBPassword +
		" dbname=" + c.DBName +
		" port=" + c.DBPort +
		" sslmode=disable TimeZone=Asia/Taipei"
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
