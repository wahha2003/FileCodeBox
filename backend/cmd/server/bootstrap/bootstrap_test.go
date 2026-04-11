package bootstrap

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/zy84338719/fileCodeBox/backend/internal/repo/db/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}

	return db
}

func TestCreateDefaultAdminCreatesUsableCredentials(t *testing.T) {
	db := newTestDB(t)

	if err := CreateDefaultAdmin(db); err != nil {
		t.Fatalf("CreateDefaultAdmin returned error: %v", err)
	}

	var admin model.User
	if err := db.Where("username = ?", defaultAdminUsername).First(&admin).Error; err != nil {
		t.Fatalf("load admin: %v", err)
	}

	if admin.Role != "admin" {
		t.Fatalf("expected admin role, got %q", admin.Role)
	}

	if admin.Status != "active" {
		t.Fatalf("expected active status, got %q", admin.Status)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(defaultAdminPassword)); err != nil {
		t.Fatalf("default admin password does not match %q: %v", defaultAdminPassword, err)
	}
}

func TestCreateDefaultAdminRepairsLegacyHash(t *testing.T) {
	db := newTestDB(t)

	admin := &model.User{
		Username:     defaultAdminUsername,
		Email:        defaultAdminEmail,
		PasswordHash: legacyDefaultAdminPasswordHash,
		Nickname:     defaultAdminNickname,
		Role:         "admin",
		Status:       "active",
	}
	if err := db.Create(admin).Error; err != nil {
		t.Fatalf("seed legacy admin: %v", err)
	}

	if err := CreateDefaultAdmin(db); err != nil {
		t.Fatalf("CreateDefaultAdmin returned error: %v", err)
	}

	var repaired model.User
	if err := db.Where("username = ?", defaultAdminUsername).First(&repaired).Error; err != nil {
		t.Fatalf("load repaired admin: %v", err)
	}

	if repaired.PasswordHash == legacyDefaultAdminPasswordHash {
		t.Fatal("expected legacy admin password hash to be repaired")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(repaired.PasswordHash), []byte(defaultAdminPassword)); err != nil {
		t.Fatalf("repaired admin password does not match %q: %v", defaultAdminPassword, err)
	}
}
