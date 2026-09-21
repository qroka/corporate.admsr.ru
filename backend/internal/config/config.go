package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds runtime settings from environment variables.
type Config struct {
	HTTPAddr   string
	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPass     string
	ASUURL     string
	ASUSecret  string
	ASUHost    string
	UploadDir  string
	SessionTTL int // hours
}

func Load() Config {
	return Config{
		HTTPAddr:   getenv("HTTP_ADDR", ":8080"),
		DBHost:     getenv("DB_HOST", "localhost"),
		DBPort:     getenv("DB_PORT", "5432"),
		DBName:     getenv("DB_NAME", "corporate_portal"),
		DBUser:     getenv("DB_USER", "myuser"),
		DBPass:     getenv("DB_PASS", ""),
		ASUURL:     getenv("ASU_URL", "https://172.17.30.42/asu_lookup.php"),
		ASUSecret:  getenv("ASU_SECRET", "asu_corporate_sync_key"),
		ASUHost:    getenv("ASU_HOST", "asu.admsr.ru"),
		UploadDir:  getenv("UPLOAD_DIR", "/var/lib/corporate-app/uploads"),
		SessionTTL: getenvInt("AUTH_SESSION_TTL_HOURS", 24),
	}
}

func (c Config) DatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.DBUser, c.DBPass, c.DBHost, c.DBPort, c.DBName,
	)
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
