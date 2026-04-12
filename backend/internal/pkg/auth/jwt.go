package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/zy84338719/fileCodeBox/backend/internal/conf"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
	jwtSecret       = []byte("FileCodeBox2025SecretKey") // 应该从配置文件读取
)

// Claims JWT claims
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken 生成 JWT token
func GenerateToken(userID uint, username, role string) (string, error) {
	expiryHours := 24 * 7
	if cfg := conf.GetGlobalConfig(); cfg != nil && cfg.User.SessionExpiryHours > 0 {
		expiryHours = cfg.User.SessionExpiryHours
	}

	claims := &Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiryHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "FileCodeBox",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(currentJWTSecret())
}

// ParseToken 解析 JWT token
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return currentJWTSecret(), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// RefreshToken 刷新 token
func RefreshToken(tokenString string) (string, error) {
	claims, err := ParseToken(tokenString)
	if err != nil {
		return "", err
	}

	// 生成新的 token
	return GenerateToken(claims.UserID, claims.Username, claims.Role)
}

// GenerateAdminToken 生成管理员 token (24小时过期)
func GenerateAdminToken(userID uint, username string) (string, error) {
	return GenerateToken(userID, username, "admin")
}

// SetJWTSecret 设置 JWT secret（从配置文件）
func SetJWTSecret(secret string) {
	jwtSecret = []byte(secret)
}

func currentJWTSecret() []byte {
	if cfg := conf.GetGlobalConfig(); cfg != nil {
		if secret := strings.TrimSpace(cfg.User.JWTSecret); secret != "" {
			return []byte(secret)
		}
	}
	return jwtSecret
}
