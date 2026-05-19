package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

type Service struct {
	key []byte
}

// New 用于创建并初始化应用实例。
// 输入参数：key 表示键名。
// 输出参数：返回 *Service, error；error 表示执行失败原因。
func New(key string) (*Service, error) {
	raw := []byte(key)
	if len(raw) < 32 {
		return nil, fmt.Errorf("encryption key must be at least 32 bytes")
	}
	return &Service{key: raw[:32]}, nil
}

// Encrypt 用于加密明文字符串。
// 输入参数：value 表示输入值。
// 输出参数：返回 string, error；error 表示执行失败原因。
func (s *Service) Encrypt(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(value), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 用于解密密文字符串。
// 输入参数：value 表示输入值。
// 输出参数：返回 string, error；error 表示执行失败原因。
func (s *Service) Decrypt(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
