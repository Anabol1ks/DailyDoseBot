package config

import (
	"os"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

type Config struct {
	DB      DBConfig
	TGtoken string
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

func Load(log *zap.Logger) *Config {
	if err := godotenv.Load(); err != nil {
		log.Warn("No .env file found, using system env")
	}

	return &Config{
		DB: DBConfig{
			Host:     getEnv("DB_HOST", log),
			Port:     getEnv("DB_PORT", log),
			User:     getEnv("DB_USER", log),
			Password: getEnv("DB_PASSWORD", log),
			Name:     getEnv("DB_NAME", log),
			SSLMode:  getEnv("DB_SSLMODE", log),
		},
		TGtoken: getEnv("TG_TOKEN", log),
	}
}

func getEnv(key string, log *zap.Logger) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	log.Error("Обязательная переменная окружения не установлена", zap.String("key", key))
	panic("missing required environment variable: " + key)
}

// func parseDuration(s string, log *zap.Logger) time.Duration {
// 	if strings.HasSuffix(s, "d") {
// 		daysStr := strings.TrimSuffix(s, "d")
// 		days, err := time.ParseDuration(daysStr + "h")
// 		if err != nil {
// 			log.Warn("Ошибка парсинга TTL", zap.Error(err))
// 			return 0
// 		}
// 		return time.Duration(24) * days
// 	}

// 	duration, err := time.ParseDuration(s)
// 	if err != nil {
// 		log.Warn("Ошибка парсинга TTL", zap.Error(err))
// 		return 0
// 	}
// 	return duration
// }
