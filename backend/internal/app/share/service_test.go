package share

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/zy84338719/fileCodeBox/backend/internal/conf"
	"github.com/zy84338719/fileCodeBox/backend/internal/repo/db"
	"github.com/zy84338719/fileCodeBox/backend/internal/repo/db/dao"
	"github.com/zy84338719/fileCodeBox/backend/internal/repo/db/model"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.GetDB()
	database, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	if err := database.AutoMigrate(&model.FileCode{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	db.SetDatabaseInstance(database)
	t.Cleanup(func() {
		db.SetDatabaseInstance(originalDB)
		sqlDB, err := database.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
}

func TestShareTextWithPasswordProtection(t *testing.T) {
	setupTestDB(t)

	svc := NewService("http://localhost", nil)
	resp, err := svc.ShareTextWithAuth(context.Background(), "hello", 1, "day", true, "secret123", nil, "127.0.0.1")
	if err != nil {
		t.Fatalf("share text: %v", err)
	}

	fileCode, err := svc.GetFileByCode(context.Background(), resp.Code)
	if err != nil {
		t.Fatalf("get file by code: %v", err)
	}

	if fileCode.AccessPasswordHash == "" {
		t.Fatal("expected access password hash to be stored")
	}

	if fileCode.AccessPasswordHash == "secret123" {
		t.Fatal("expected access password to be hashed")
	}

	if err := svc.ValidateAccess(fileCode, "secret123"); err != nil {
		t.Fatalf("validate correct password: %v", err)
	}

	if err := svc.ValidateAccess(fileCode, "wrong-password"); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("expected invalid password error, got %v", err)
	}

	if err := svc.ValidateAccess(fileCode, ""); !errors.Is(err, ErrPasswordRequired) {
		t.Fatalf("expected password required error, got %v", err)
	}
}

func TestShareTextRejectsMissingPasswordWhenProtectionEnabled(t *testing.T) {
	setupTestDB(t)

	svc := NewService("http://localhost", nil)
	_, err := svc.ShareTextWithAuth(context.Background(), "hello", 1, "day", true, "", nil, "127.0.0.1")
	if !errors.Is(err, ErrPasswordRequired) {
		t.Fatalf("expected password required error, got %v", err)
	}
}

func TestValidateAccessRejectsMisconfiguredProtectedShare(t *testing.T) {
	setupTestDB(t)

	repo := dao.NewFileCodeRepository()
	fileCode := &model.FileCode{
		Code:         "legacy123",
		Text:         "legacy share",
		ExpiredCount: 1,
		RequireAuth:  true,
	}
	if err := repo.Create(context.Background(), fileCode); err != nil {
		t.Fatalf("create legacy share: %v", err)
	}

	svc := NewService("http://localhost", nil)
	loaded, err := svc.GetFileByCode(context.Background(), fileCode.Code)
	if err != nil {
		t.Fatalf("load legacy share: %v", err)
	}

	if err := svc.ValidateAccess(loaded, "anything"); !errors.Is(err, ErrPasswordNotConfigured) {
		t.Fatalf("expected password not configured error, got %v", err)
	}
}

func TestGenerateCodeUsesConfiguredLengthAndCharset(t *testing.T) {
	setupTestDB(t)

	originalConfig := conf.GetGlobalConfig()
	conf.SetGlobalConfig(&conf.AppConfiguration{
		Upload: conf.UploadConfig{
			ShareCodeLength:  6,
			ShareCodeCharset: "AB0123",
		},
	})
	t.Cleanup(func() {
		conf.SetGlobalConfig(originalConfig)
	})

	svc := NewService("http://localhost", nil)
	code := svc.GenerateCode()
	if len(code) != 6 {
		t.Fatalf("expected code length 6, got %q (%d)", code, len(code))
	}

	for _, ch := range code {
		if !strings.ContainsRune("AB0123", ch) {
			t.Fatalf("expected code %q to use configured charset only", code)
		}
	}
}

func TestGetFileByCodeFallsBackToUppercaseLookupWhenConfigured(t *testing.T) {
	setupTestDB(t)

	originalConfig := conf.GetGlobalConfig()
	conf.SetGlobalConfig(&conf.AppConfiguration{
		Upload: conf.UploadConfig{
			ShareCodeLength:  4,
			ShareCodeCharset: "AB0123456789",
		},
	})
	t.Cleanup(func() {
		conf.SetGlobalConfig(originalConfig)
	})

	repo := dao.NewFileCodeRepository()
	fileCode := &model.FileCode{
		Code:         "AB12",
		Text:         "uppercase share",
		ExpiredCount: 1,
	}
	if err := repo.Create(context.Background(), fileCode); err != nil {
		t.Fatalf("create uppercase share: %v", err)
	}

	svc := NewService("http://localhost", nil)
	loaded, err := svc.GetFileByCode(context.Background(), "ab12")
	if err != nil {
		t.Fatalf("lookup share with lowercase input: %v", err)
	}

	if loaded.Code != "AB12" {
		t.Fatalf("expected AB12, got %q", loaded.Code)
	}
}
