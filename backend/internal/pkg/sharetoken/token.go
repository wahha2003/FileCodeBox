package sharetoken

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const cookiePrefix = "fcb_share_access_"

var (
	ErrMissingToken  = errors.New("share access token missing")
	ErrInvalidToken  = errors.New("share access token invalid")
	ErrExpiredToken  = errors.New("share access token expired")
	ErrSecretMissing = errors.New("share access token secret missing")
)

type Claims struct {
	Code      string `json:"code"`
	ExpiresAt int64  `json:"exp"`
}

func CookieName(code string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(code)))
	return cookiePrefix + hex.EncodeToString(sum[:8])
}

func Sign(code string, expiresAt time.Time, secret string) (string, error) {
	code = strings.TrimSpace(code)
	secret = strings.TrimSpace(secret)

	if code == "" || secret == "" {
		return "", ErrSecretMissing
	}

	claims := Claims{
		Code:      code,
		ExpiresAt: expiresAt.UTC().Unix(),
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal share access token: %w", err)
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := sign(encodedPayload, secret)
	return encodedPayload + "." + signature, nil
}

func Verify(token string, expectedCode string, secret string, now time.Time) (*Claims, error) {
	token = strings.TrimSpace(token)
	expectedCode = strings.TrimSpace(expectedCode)
	secret = strings.TrimSpace(secret)

	if token == "" {
		return nil, ErrMissingToken
	}
	if secret == "" {
		return nil, ErrSecretMissing
	}

	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, ErrInvalidToken
	}

	payload, providedSignature := parts[0], parts[1]
	expectedSignature := sign(payload, secret)
	if !hmac.Equal([]byte(providedSignature), []byte(expectedSignature)) {
		return nil, ErrInvalidToken
	}

	decodedPayload, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, ErrInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(decodedPayload, &claims); err != nil {
		return nil, ErrInvalidToken
	}

	if claims.Code == "" || claims.ExpiresAt <= 0 {
		return nil, ErrInvalidToken
	}
	if expectedCode != "" && claims.Code != expectedCode {
		return nil, ErrInvalidToken
	}
	if !time.Unix(claims.ExpiresAt, 0).After(now.UTC()) {
		return nil, ErrExpiredToken
	}

	return &claims, nil
}

func sign(payload string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
