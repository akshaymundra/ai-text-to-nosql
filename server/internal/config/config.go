package config

import (
	"os"
	"sync"
)

type Config struct {
	ServerPort string
	AppEnv     string
}

var (
	instance *Config
	once     sync.Once
)

func Init() {
	once.Do(func() {
		instance = &Config{
			ServerPort: getEnv("SERVER_PORT", "3000"),
			AppEnv:     getEnv("APP_ENV", "development"),
		}
	})
}

func GetInstance() *Config {
	return instance
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
