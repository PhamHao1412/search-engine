package app

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                  string
	Env                   string
	DBHost                string
	DBPort                string
	DBUser                string
	DBPassword            string
	DBName                string
	DBSchema              string
	DBSSLMode             string
	RedisHost             string
	RedisPort             string
	RedisPassword         string
	OpenSearchURL         string
	KafkaBrokers          string
	GoogleTranslateAPIKey string
	OpenAIAPIKey          string
	OpenAIModel           string
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load(".env")

	cfg := &Config{
		Port:                  getEnv("PORT", "8081"),
		Env:                   getEnv("ENV", "development"),
		DBHost:                getEnv("DB_HOST", "localhost"),
		DBPort:                getEnv("DB_PORT", "5432"),
		DBUser:                getEnv("DB_USER", "postgres"),
		DBPassword:            getEnv("DB_PASSWORD", "postgrespassword"),
		DBName:                getEnv("DB_NAME", "swiftsearch_search"),
		DBSchema:              getEnv("DB_SCHEMA", "search"),
		DBSSLMode:             getEnv("DB_SSLMODE", "disable"),
		RedisHost:             getEnv("REDIS_HOST", "localhost"),
		RedisPort:             getEnv("REDIS_PORT", "6379"),
		RedisPassword:         getEnv("REDIS_PASSWORD", ""),
		OpenSearchURL:         getEnv("OPENSEARCH_URL", "http://localhost:9200"),
		KafkaBrokers:          getEnv("KAFKA_BROKERS", "localhost:29092"),
		GoogleTranslateAPIKey: getEnv("GOOGLE_TRANSLATE_API_KEY", ""),
		OpenAIAPIKey:          getEnv("OPENAI_API_KEY", ""),
		OpenAIModel:           getEnv("OPENAI_MODEL", "gpt-4o-mini"),
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
