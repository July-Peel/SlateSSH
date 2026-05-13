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

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

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


