package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBPort     string
	DBHost     string
	DBUser     string
	DBPassword string
	DBName     string
}

var Envs = initialConfig()

func initialConfig() Config {
	godotenv.Load()

	return Config{
		DBPort:     getEnv("DB_PORT", "8080"),
		DBHost:     getEnv("DB_HOST", "usuario"),
		DBUser:     getEnv("DB_USER", "usuario"),
		DBPassword: getEnv("DB_PASSWORD", "usuario"),
		DBName:     getEnv("DB_NAME", "usuario"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}
