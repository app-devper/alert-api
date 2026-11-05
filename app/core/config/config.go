package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port      string
	MongoHost string
	DbPrefix  string
	RedisHost string
	SecretKey string
	System    string
	ClientId  string
}

func MustLoad() *Config {
	if err := godotenv.Load(".env"); err != nil {
		log.Print(err)
	}
	return &Config{
		Port:      mustEnv("PORT"),
		MongoHost: mustEnv("MONGO_HOST"),
		DbPrefix:  DbPrefixFromEnv(),
		RedisHost: mustEnv("REDIS_HOST"),
		SecretKey: mustEnv("SECRET_KEY"),
		System:    mustEnv("SYSTEM"),
		ClientId:  os.Getenv("CLIENT_ID"),
	}
}

// DbPrefixFromEnv is the tenant database prefix this deployment uses; the
// service and its maintenance commands must agree on it.
func DbPrefixFromEnv() string {
	return envOr("MONGO_DB_PREFIX", "alert")
}

func mustEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("missing required env: %s", key)
	}
	return value
}

func envOr(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
