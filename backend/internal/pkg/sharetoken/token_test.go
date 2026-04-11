package sharetoken

import (
	"errors"
	"testing"
	"time"
)

func TestSignAndVerify(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	expiresAt := now.Add(30 * time.Minute)

	token, err := Sign("1234", expiresAt, "secret-key")
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	claims, err := Verify(token, "1234", "secret-key", now)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}

	if claims.Code != "1234" {
		t.Fatalf("unexpected code: %s", claims.Code)
	}
	if claims.ExpiresAt != expiresAt.Unix() {
		t.Fatalf("unexpected expires at: %d", claims.ExpiresAt)
	}
}

func TestVerifyRejectsExpiredToken(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	token, err := Sign("1234", now.Add(-time.Minute), "secret-key")
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	_, err = Verify(token, "1234", "secret-key", now)
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("expected expired token error, got %v", err)
	}
}

func TestVerifyRejectsTamperedToken(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	token, err := Sign("1234", now.Add(time.Hour), "secret-key")
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	token += "tampered"
	_, err = Verify(token, "1234", "secret-key", now)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected invalid token error, got %v", err)
	}
}
