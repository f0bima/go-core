package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds the base configuration shared by all services.
type Config struct {
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	ServiceName string
	Port        string
}

// LoadConfig loads configuration from .env file and environment variables.
func LoadConfig(extraEnvPaths ...string) *Config {
	paths := append([]string{".env"}, extraEnvPaths...)
	err := godotenv.Load(paths...)
	if err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	return &Config{
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "5432"),
		DBUser:      getEnv("DB_USER", "postgres"),
		DBPassword:  getEnv("DB_PASSWORD", "postgres"),
		DBName:      getEnv("DB_NAME", "app_db"),
		ServiceName: getEnv("SERVICE_NAME", "app-service"),
		Port:        getEnv("PORT", "8080"),
	}
}

func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
