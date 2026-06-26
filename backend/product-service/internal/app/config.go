package app

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port         string
	Env          string
	DBHost       string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string
	DBSchema     string
	DBSSLMode    string
	KafkaBrokers string
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load(".env")

	cfg := &Config{
		Port:         getEnv("PORT", "8080"),
		Env:          getEnv("ENV", "development"),
		DBHost:       getEnv("DB_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "5432"),
		DBUser:       getEnv("DB_USER", "postgres"),
		DBPassword:   getEnv("DB_PASSWORD", "postgrespassword"),
		DBName:       getEnv("DB_NAME", "amaze_search"),
		DBSchema:     getEnv("DB_SCHEMA", "product"),
		DBSSLMode:    getEnv("DB_SSLMODE", "disable"),
		KafkaBrokers: getEnv("KAFKA_BROKERS", "localhost:29092"),
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
