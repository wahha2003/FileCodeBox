package auth

import (
	"testing"
	"time"

	"github.com/zy84338719/fileCodeBox/backend/internal/conf"
)

func TestGenerateTokenUsesConfiguredSessionExpiryHours(t *testing.T) {
	originalConfig := conf.GetGlobalConfig()
	conf.SetGlobalConfig(&conf.AppConfiguration{
		User: conf.UserConfig{
			SessionExpiryHours: 1,
			JWTSecret:          "test-secret",
		},
	})
	t.Cleanup(func() {
		conf.SetGlobalConfig(originalConfig)
	})

	token, err := GenerateToken(1, "alice", "user")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}

	ttl := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time)
	if ttl < 59*time.Minute || ttl > 61*time.Minute {
		t.Fatalf("expected token ttl around 1 hour, got %s", ttl)
	}
}
