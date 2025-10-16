package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Database DatabaseConfig
	Server   ServerConfig
	Monitor  MonitorConfig
	Logger   LoggerConfig
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type ServerConfig struct {
	Host string
	Port int
}

type MonitorConfig struct {
	CheckInterval     time.Duration
	ConnectionTimeout time.Duration
	MaxRetryAttempts  int
}

type LoggerConfig struct {
	Level  string
	Format string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		// It's okay if .env file doesn't exist in production
	}

	port, _ := strconv.Atoi(getEnv("DB_PORT", "5432"))
	serverPort, _ := strconv.Atoi(getEnv("SERVER_PORT", "4622"))
	maxRetry, _ := strconv.Atoi(getEnv("MAX_RETRY_ATTEMPTS", "5"))

	checkInterval, _ := time.ParseDuration(getEnv("BOOTSTRAP_CHECK_INTERVAL", "24h"))
	connTimeout, _ := time.ParseDuration(getEnv("CONNECTION_TIMEOUT", "30s"))

	return &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     port,
			User:     getEnv("DB_USER", "pactus_user"),
			Password: getEnv("DB_PASSWORD", "pactus_password"),
			DBName:   getEnv("DB_NAME", "pactus_tracker"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: serverPort,
		},
		Monitor: MonitorConfig{
			CheckInterval:     checkInterval,
			ConnectionTimeout: connTimeout,
			MaxRetryAttempts:  maxRetry,
		},
		Logger: LoggerConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
