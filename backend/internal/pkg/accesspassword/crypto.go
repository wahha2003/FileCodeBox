package accesspassword

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/zy84338719/fileCodeBox/backend/internal/conf"
)

const fallbackSecret = "FileCodeBox2025AccessPasswordSecret"

func Encrypt(plainText string) (string, error) {
	if plainText == "" {
		return "", nil
	}

	block, err := aes.NewCipher(secretKey())
	if err != nil {
		return "", fmt.Errorf("create access password cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create access password gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate access password nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plainText), nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

func Decrypt(cipherText string) (string, error) {
	if strings.TrimSpace(cipherText) == "" {
		return "", nil
	}

	payload, err := base64.RawURLEncoding.DecodeString(cipherText)
	if err != nil {
		return "", fmt.Errorf("decode access password: %w", err)
	}

	block, err := aes.NewCipher(secretKey())
	if err != nil {
		return "", fmt.Errorf("create access password cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create access password gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(payload) < nonceSize {
		return "", fmt.Errorf("invalid access password payload")
	}

	nonce, encrypted := payload[:nonceSize], payload[nonceSize:]
	plainText, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt access password: %w", err)
	}

	return string(plainText), nil
}

func secretKey() []byte {
	secret := fallbackSecret
	if cfg := conf.GetGlobalConfig(); cfg != nil {
		if configured := strings.TrimSpace(cfg.User.JWTSecret); configured != "" {
			secret = configured
		}
	}

	sum := sha256.Sum256([]byte("share-password:" + secret))
	return sum[:]
}
