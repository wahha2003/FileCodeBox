package filecleanup

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/zy84338719/fileCodeBox/backend/internal/repo/db"
	"github.com/zy84338719/fileCodeBox/backend/internal/repo/db/dao"
	"github.com/zy84338719/fileCodeBox/backend/internal/repo/db/model"
	"github.com/zy84338719/fileCodeBox/backend/internal/storage"
	"gorm.io/gorm"
)

type fakeStorage struct {
	exists     bool
	deleteErr  error
	deleted    []string
	existsSeen []string
}

func (f *fakeStorage) SaveFile(ctx context.Context, file *multipart.FileHeader, savePath string) (*storage.FileOperationResult, error) {
	panic("not implemented")
}

func (f *fakeStorage) DeleteFile(ctx context.Context, filePath string) error {
	f.deleted = append(f.deleted, filePath)
	return f.deleteErr
}

func (f *fakeStorage) GetFile(ctx context.Context, filePath string) ([]byte, error) {
	panic("not implemented")
}

func (f *fakeStorage) FileExists(ctx context.Context, filePath string) bool {
	f.existsSeen = append(f.existsSeen, filePath)
	return f.exists
}

func (f *fakeStorage) SaveChunk(ctx context.Context, uploadID string, chunkIndex int, data []byte) error {
	panic("not implemented")
}

func (f *fakeStorage) MergeChunks(ctx context.Context, uploadID string, totalChunks int, savePath string) error {
	panic("not implemented")
}

func (f *fakeStorage) CleanChunks(ctx context.Context, uploadID string) error {
	panic("not implemented")
}

func (f *fakeStorage) GetFileSize(ctx context.Context, filePath string) (int64, error) {
	panic("not implemented")
}

func (f *fakeStorage) GetFileURL(ctx context.Context, filePath string) (string, error) {
	panic("not implemented")
}

func (f *fakeStorage) GetFileReader(ctx context.Context, filePath string) (io.ReadCloser, int64, error) {
	panic("not implemented")
}

type fakeUserStats struct {
	updates []int64
}

func (f *fakeUserStats) UpdateUserStats(userID uint, statsType string, value int64) error {
	f.updates = append(f.updates, value)
	return nil
}

func setupCleanupTestDB(t *testing.T) {
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

func TestDeleteFileCodeDeletesStoredObjectBeforeDeletingRecord(t *testing.T) {
	setupCleanupTestDB(t)

	repo := dao.NewFileCodeRepository()
	userID := uint(7)
	fileCode := &model.FileCode{
		Code:         "A100",
		FilePath:     "uploads/2026/04/12",
		UUIDFileName: "remote.bin",
		Size:         128,
		UserID:       &userID,
	}
	if err := repo.Create(context.Background(), fileCode); err != nil {
		t.Fatalf("create file: %v", err)
	}

	fakeStore := &fakeStorage{exists: true}
	userStats := &fakeUserStats{}
	service := NewService(func() storage.StorageInterface { return fakeStore }, userStats)

	if err := service.DeleteFileCode(context.Background(), fileCode); err != nil {
		t.Fatalf("delete file code: %v", err)
	}

	expectedPath := filepath.Join("uploads", "2026", "04", "12", "remote.bin")
	if len(fakeStore.deleted) != 1 || fakeStore.deleted[0] != expectedPath {
		t.Fatalf("expected remote object delete to be called once, got %#v", fakeStore.deleted)
	}

	if _, err := repo.GetByCode(context.Background(), fileCode.Code); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected db record deleted, got %v", err)
	}

	if len(userStats.updates) != 1 || userStats.updates[0] != -128 {
		t.Fatalf("expected storage quota released once, got %#v", userStats.updates)
	}
}

func TestDeleteFileCodeKeepsRecordWhenStoredObjectDeletionFails(t *testing.T) {
	setupCleanupTestDB(t)

	repo := dao.NewFileCodeRepository()
	fileCode := &model.FileCode{
		Code:         "A101",
		FilePath:     "uploads/2026/04/12",
		UUIDFileName: "remote.bin",
		Size:         128,
	}
	if err := repo.Create(context.Background(), fileCode); err != nil {
		t.Fatalf("create file: %v", err)
	}

	fakeStore := &fakeStorage{
		exists:    true,
		deleteErr: errors.New("remote delete failed"),
	}
	service := NewService(func() storage.StorageInterface { return fakeStore }, nil)

	if err := service.DeleteFileCode(context.Background(), fileCode); err == nil {
		t.Fatal("expected delete file code to fail when remote delete fails")
	}

	if _, err := repo.GetByCode(context.Background(), fileCode.Code); err != nil {
		t.Fatalf("expected db record kept for retry, got %v", err)
	}
}
