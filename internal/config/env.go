package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DBPort                 string
	DBHost                 string
	DBUser                 string
	DBPassword             string
	DBName                 string
	JWTExpirationInSeconds int64
	JWTSecret              string
}

var Envs = initialConfig()

func initialConfig() Config {
	godotenv.Load()

	return Config{
		DBPort:                 getEnv("DB_PORT", "8080"),
		DBHost:                 getEnv("DB_HOST", "usuario"),
		DBUser:                 getEnv("DB_USER", "usuario"),
		DBPassword:             getEnv("DB_PASSWORD", "usuario"),
		DBName:                 getEnv("DB_NAME", "usuario"),
		JWTSecret:              getEnv("JWT_SECRET", "not_a_secret"),
		JWTExpirationInSeconds: getEnvAsInt("JWT_EXP", 3600*24*7),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}

func getEnvAsInt(key string, fallback int64) int64 {
	if value, ok := os.LookupEnv(key); ok {
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fallback
		}

		return i
	}

	return fallback
}
