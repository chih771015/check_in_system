package config

import (
	"os"
	"strconv"
)

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
