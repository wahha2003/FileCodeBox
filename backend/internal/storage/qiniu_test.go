package storage

import (
	"context"
	"strings"
	"testing"
)

func TestQiniuGetFileURLPrivateAddsSchemeForBareDomain(t *testing.T) {
	storage := NewQiniuStorage(&StorageConfig{
		QiniuAccessKey: "test-ak",
		QiniuSecretKey: "test-sk",
		QiniuBucket:    "test-bucket",
		QiniuDomain:    "cdn.example.com",
		QiniuUseHTTPS:  true,
		QiniuPrivate:   true,
	})

	got, err := storage.GetFileURL(context.Background(), "uploads/demo file.txt")
	if err != nil {
		t.Fatalf("GetFileURL returned error: %v", err)
	}
	if !strings.HasPrefix(got, "https://cdn.example.com/uploads/demo%20file.txt?e=") {
		t.Fatalf("unexpected signed url: %s", got)
	}
	if !strings.Contains(got, "&token=") {
		t.Fatalf("signed url missing token: %s", got)
	}
}

func TestQiniuGetFileURLPublicKeepsConfiguredScheme(t *testing.T) {
	storage := NewQiniuStorage(&StorageConfig{
		QiniuAccessKey: "test-ak",
		QiniuSecretKey: "test-sk",
		QiniuBucket:    "test-bucket",
		QiniuDomain:    "http://cdn.example.com/",
		QiniuUseHTTPS:  true,
	})

	got, err := storage.GetFileURL(context.Background(), "uploads/test.exe")
	if err != nil {
		t.Fatalf("GetFileURL returned error: %v", err)
	}
	if got != "http://cdn.example.com/uploads/test.exe" {
		t.Fatalf("unexpected public url: %s", got)
	}
}
