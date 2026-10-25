package config

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort     string
	AppEnv         string
	DatabaseURL    string
	DBPingTimeout  time.Duration
	JWTSecret      string
	JWTTTL         time.Duration
	OllamaURL      string
	OllamaTimeout  time.Duration
	EmbeddingModel string
}

const EnvDevelopment = "development"

var requiredEnvs = []string{
	"DATABASE_URL",
	"JWT_SECRET",
}

var (
	instance *Config
	once     sync.Once
)

func Init() {
	once.Do(func() {
		_ = godotenv.Load()
		instance = &Config{
			ServerPort:     getEnv("SERVER_PORT", "3000"),
			AppEnv:         getEnv("APP_ENV", EnvDevelopment),
			DatabaseURL:    getEnv("DATABASE_URL", ""),
			DBPingTimeout:  getDurationEnv("DB_PING_TIMEOUT", 5*time.Second),
			JWTSecret:      os.Getenv("JWT_SECRET"),
			JWTTTL:         getDurationEnv("JWT_TTL", 24*time.Hour),
			OllamaURL:      getEnv("OLLAMA_URL", "http://localhost:11434"),
			OllamaTimeout:  getDurationEnv("OLLAMA_TIMEOUT", 60*time.Second),
			EmbeddingModel: getEnv("EMBEDDING_MODEL", "qwen3-embedding:0.6b"),
		}
	})
}

func Validate() error {
	var missing []string
	for _, key := range requiredEnvs {
		if os.Getenv(key) == "" {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required env: %s", strings.Join(missing, ", "))
	}
	return nil
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

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return d
}
