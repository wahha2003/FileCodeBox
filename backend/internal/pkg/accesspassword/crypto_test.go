package accesspassword

import "testing"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	cipherText, err := Encrypt("demo-password")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if cipherText == "" {
		t.Fatal("Encrypt() returned empty ciphertext")
	}

	plainText, err := Decrypt(cipherText)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if plainText != "demo-password" {
		t.Fatalf("Decrypt() = %q, want %q", plainText, "demo-password")
	}
}
