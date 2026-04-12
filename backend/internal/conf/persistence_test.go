package conf

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSaveConfigToFilePreservesUnrelatedSections(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("ui:\n  theme: custom-theme\n"), 0644); err != nil {
		t.Fatalf("write seed config: %v", err)
	}

	cfg := &AppConfiguration{
		Server: ServerConfig{
			Host:          "0.0.0.0",
			Port:          23456,
			Mode:          "release",
			BaseURL:       "https://api.example.com",
			PublicBaseURL: "https://example.com",
			ReadTimeout:   30,
			WriteTimeout:  30,
		},
		App: AppConfig{
			Name:        "FileCodeBox",
			Version:     "1.0.0",
			Description: "test",
			DataPath:    "./data",
			Production:  true,
		},
		User: UserConfig{
			AllowUserRegistration: false,
			RequireEmailVerify:    false,
			UserUploadSize:        50 * 1024 * 1024,
			UserStorageQuota:      2 * 1024 * 1024 * 1024,
			SessionExpiryHours:    12,
			MaxSessionsPerUser:    5,
			JWTSecret:             "secret",
		},
		Upload: UploadConfig{
			OpenUpload:       true,
			UploadSize:       20 * 1024 * 1024,
			EnableChunk:      true,
			ChunkSize:        5 * 1024 * 1024,
			MaxSaveSeconds:   0,
			RequireLogin:     false,
			ShareCodeLength:  6,
			ShareCodeCharset: "AB0123",
		},
		Storage: StorageConfig{
			Type:        "local",
			StoragePath: "./data/uploads",
		},
	}

	if err := SaveConfigToFile(cfg, configPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	var stored struct {
		UI struct {
			Theme string `yaml:"theme"`
		} `yaml:"ui"`
		Server struct {
			Port int `yaml:"port"`
		} `yaml:"server"`
		Upload struct {
			UploadSize      int64 `yaml:"upload_size"`
			ShareCodeLength int   `yaml:"share_code_length"`
		} `yaml:"upload"`
		User struct {
			SessionExpiryHours int `yaml:"session_expiry_hours"`
		} `yaml:"user"`
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read stored config: %v", err)
	}
	if err := yaml.Unmarshal(content, &stored); err != nil {
		t.Fatalf("unmarshal stored config: %v", err)
	}

	if stored.UI.Theme != "custom-theme" {
		t.Fatalf("expected UI theme to be preserved, got %q", stored.UI.Theme)
	}
	if stored.Server.Port != 23456 {
		t.Fatalf("expected server port 23456, got %d", stored.Server.Port)
	}
	if stored.Upload.UploadSize != 20*1024*1024 {
		t.Fatalf("expected upload size to be written, got %d", stored.Upload.UploadSize)
	}
	if stored.Upload.ShareCodeLength != 6 {
		t.Fatalf("expected share code length 6, got %d", stored.Upload.ShareCodeLength)
	}
	if stored.User.SessionExpiryHours != 12 {
		t.Fatalf("expected session expiry hours 12, got %d", stored.User.SessionExpiryHours)
	}
}
