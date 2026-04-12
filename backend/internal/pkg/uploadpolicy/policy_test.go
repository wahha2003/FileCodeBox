package uploadpolicy

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/zy84338719/fileCodeBox/backend/internal/conf"
	"github.com/zy84338719/fileCodeBox/backend/internal/repo/db"
	"github.com/zy84338719/fileCodeBox/backend/internal/repo/db/dao"
	"github.com/zy84338719/fileCodeBox/backend/internal/repo/db/model"
	"gorm.io/gorm"
)

func setupUserDB(t *testing.T) {
	t.Helper()

	originalDB := db.GetDB()
	database, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	if err := database.AutoMigrate(&model.User{}); err != nil {
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

func TestResolveRejectsAnonymousUploadWhenDisabled(t *testing.T) {
	originalConfig := conf.GetGlobalConfig()
	conf.SetGlobalConfig(&conf.AppConfiguration{
		Upload: conf.UploadConfig{
			OpenUpload:   false,
			RequireLogin: false,
		},
	})
	t.Cleanup(func() {
		conf.SetGlobalConfig(originalConfig)
	})

	_, err := Resolve(context.Background(), nil)
	if !errors.Is(err, ErrAnonymousUploadDisabled) {
		t.Fatalf("expected anonymous upload disabled error, got %v", err)
	}
}

func TestResolveUsesUserUploadAndQuotaLimits(t *testing.T) {
	setupUserDB(t)

	originalConfig := conf.GetGlobalConfig()
	conf.SetGlobalConfig(&conf.AppConfiguration{
		Upload: conf.UploadConfig{
			OpenUpload: true,
			UploadSize: 10 * 1024 * 1024,
		},
		User: conf.UserConfig{
			UserUploadSize:   50 * 1024 * 1024,
			UserStorageQuota: 100 * 1024 * 1024,
		},
	})
	t.Cleanup(func() {
		conf.SetGlobalConfig(originalConfig)
	})

	repo := dao.NewUserRepository()
	user := &model.User{
		Username:        "alice",
		Email:           "alice@example.com",
		PasswordHash:    "hash",
		TotalStorage:    60 * 1024 * 1024,
		MaxUploadSize:   20 * 1024 * 1024,
		MaxStorageQuota: 64 * 1024 * 1024,
	}
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	userID := user.ID
	policy, err := Resolve(context.Background(), &userID)
	if err != nil {
		t.Fatalf("resolve policy: %v", err)
	}

	if policy.MaxUploadSize != 20*1024*1024 {
		t.Fatalf("expected per-user upload limit, got %d", policy.MaxUploadSize)
	}
	if policy.StorageQuota != 64*1024*1024 {
		t.Fatalf("expected per-user quota, got %d", policy.StorageQuota)
	}
	if err := policy.ValidateFileSize(21 * 1024 * 1024); !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("expected file too large error, got %v", err)
	}
	if err := policy.ValidateStorageQuota(5 * 1024 * 1024); !errors.Is(err, ErrStorageQuotaExceeded) {
		t.Fatalf("expected storage quota exceeded error, got %v", err)
	}
}
