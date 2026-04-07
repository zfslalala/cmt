package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Config struct {
	APIKey      string
	APIBase     string
	Model       string
	Temperature float64
	MaxTokens   int
}

func Load() (*Config, error) {
	// 加载 .env 文件
	loadEnvFiles()

	cfg := &Config{
		APIKey:      getEnv("CMT_API_KEY", ""),
		APIBase:     getEnv("CMT_API_URL", "https://api.openai.com/v1"),
		Model:       getEnv("CMT_MODEL", "gpt-4o-mini"),
		Temperature: 0.7,
		MaxTokens:   1000,
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("CMT_API_KEY 未设置，请在环境变量或 .env 文件中配置")
	}

	return cfg, nil
}

func loadEnvFiles() {
	// .env 文件路径列表（按优先级从低到高）
	// 后续加载的文件会覆盖之前的值，但不会覆盖已存在的系统环境变量
	locations := []string{
		filepath.Join(os.Getenv("HOME"), ".cmt.env"), // 用户 home 目录
		filepath.Join(os.Getenv("HOME"), ".env"),     // 用户 home 目录
		".env", // 当前目录
	}

	for _, loc := range locations {
		// godotenv.Load 不会覆盖已存在的环境变量
		// 因此系统环境变量优先
		_ = godotenv.Load(loc)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
