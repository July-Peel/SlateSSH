package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	Host          string
	Port          int
	DataDir       string
	DBPath        string
	SessionSecret string
	EncryptionKey string
	FrontendDir   string
	GuacdHost     string
	GuacdPort     int
}

// Load 用于读取运行配置并准备持久化密钥。
// 输入参数：无。
// 输出参数：返回 Config, error；error 表示执行失败原因。
func Load() (Config, error) {
	dataDir := getenv("STACK_GO_DATA_DIR", filepath.Join(".", "data"))
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return Config{}, err
	}

	port, err := strconv.Atoi(getenv("PORT", "3210"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid PORT: %w", err)
	}
	guacdPort, err := strconv.Atoi(getenv("GUACD_PORT", "4822"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid GUACD_PORT: %w", err)
	}

	sessionSecret, err := ensurePersistentSecret("SESSION_SECRET", filepath.Join(dataDir, "session_secret"), 32)
	if err != nil {
		return Config{}, err
	}
	encryptionKey, err := ensurePersistentSecret("ENCRYPTION_KEY", filepath.Join(dataDir, "encryption_key"), 32)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Host:          getenv("HOST", "0.0.0.0"),
		Port:          port,
		DataDir:       dataDir,
		DBPath:        filepath.Join(dataDir, "slatessh.db"),
		SessionSecret: sessionSecret,
		EncryptionKey: encryptionKey,
		FrontendDir:   filepath.Join("..", "frontend"),
		GuacdHost:     getenv("GUACD_HOST", "guacd"),
		GuacdPort:     guacdPort,
	}, nil
}

// getenv 用于读取环境变量并在为空时返回默认值。
// 输入参数：key 表示键名；fallback 表示默认值。
// 输出参数：返回 string。
func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// ensurePersistentSecret 用于获取或生成需要持久保存的随机密钥。
// 输入参数：key 表示键名；path 表示路径；size 表示字节长度。
// 输出参数：返回 string, error；error 表示执行失败原因。
func ensurePersistentSecret(key, path string, size int) (string, error) {
	if value := os.Getenv(key); value != "" {
		return value, nil
	}
	if data, err := os.ReadFile(path); err == nil {
		value := string(data)
		if len(value) >= size {
			_ = os.Setenv(key, value)
			return value, nil
		}
	}

	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	value := hex.EncodeToString(buffer)
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
		return "", err
	}
	_ = os.Setenv(key, value)
	return value, nil
}
