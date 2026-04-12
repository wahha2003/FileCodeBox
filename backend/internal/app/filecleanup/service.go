package filecleanup

import (
	"context"
	"fmt"
	"strings"

	"github.com/zy84338719/fileCodeBox/backend/internal/conf"
	"github.com/zy84338719/fileCodeBox/backend/internal/pkg/logger"
	"github.com/zy84338719/fileCodeBox/backend/internal/repo/db/dao"
	"github.com/zy84338719/fileCodeBox/backend/internal/repo/db/model"
	"github.com/zy84338719/fileCodeBox/backend/internal/storage"
	"go.uber.org/zap"
)

// UserStatsUpdater 用于回收用户存储占用。
type UserStatsUpdater interface {
	UpdateUserStats(userID uint, statsType string, value int64) error
}

// Service 负责统一清理分享记录及其对应的物理文件。
type Service struct {
	fileCodeRepo   *dao.FileCodeRepository
	storageFactory func() storage.StorageInterface
	userService    UserStatsUpdater
}

func NewService(storageFactory func() storage.StorageInterface, userService UserStatsUpdater) *Service {
	return &Service{
		fileCodeRepo:   dao.NewFileCodeRepository(),
		storageFactory: storageFactory,
		userService:    userService,
	}
}

func (s *Service) SetUserService(userService UserStatsUpdater) {
	s.userService = userService
}

// HasExpiredFiles 使用轻量查询判断是否存在待清理记录。
func (s *Service) HasExpiredFiles(ctx context.Context) (bool, error) {
	return s.fileCodeRepo.HasExpiredFiles(ctx)
}

// CleanupExpiredFiles 扫描并删除所有已过期的分享记录。
func (s *Service) CleanupExpiredFiles(ctx context.Context) (int64, int64, error) {
	expiredFiles, err := s.fileCodeRepo.GetExpiredFiles(ctx)
	if err != nil {
		return 0, 0, err
	}

	var deletedCount int64
	var freedSpace int64
	for _, fileCode := range expiredFiles {
		if err := s.DeleteFileCode(ctx, fileCode); err != nil {
			logWarn(
				"Failed to cleanup expired file",
				zap.String("code", fileCode.Code),
				zap.Uint("file_id", fileCode.ID),
				zap.Error(err),
			)
			continue
		}

		deletedCount++
		freedSpace += fileCode.Size
	}

	return deletedCount, freedSpace, nil
}

// CleanupIfExpired 在单条分享已过期时立即删除它。
func (s *Service) CleanupIfExpired(ctx context.Context, fileCode *model.FileCode) (bool, error) {
	if fileCode == nil || !fileCode.IsExpired() {
		return false, nil
	}

	return true, s.DeleteFileCode(ctx, fileCode)
}

// DeleteFileCode 删除分享记录及其对应的物理文件。
func (s *Service) DeleteFileCode(ctx context.Context, fileCode *model.FileCode) error {
	if fileCode == nil {
		return nil
	}

	if err := s.deleteStoredFile(ctx, fileCode); err != nil {
		return err
	}

	if err := s.fileCodeRepo.DeleteByFileCode(ctx, fileCode); err != nil {
		return err
	}

	s.releaseUserStorage(fileCode)
	return nil
}

func (s *Service) deleteStoredFile(ctx context.Context, fileCode *model.FileCode) error {
	if fileCode == nil || fileCode.FilePath == "" {
		return nil
	}

	filePath := fileCode.GetFilePath()
	if filePath == "" {
		return nil
	}

	storageSvc := s.storageForFile(fileCode)
	if storageSvc == nil {
		return fmt.Errorf("storage is unavailable for deleting %s", filePath)
	}

	// 轻量存在性检查，避免对象已经不存在时把重删除当成失败。
	if !storageSvc.FileExists(ctx, filePath) {
		return nil
	}

	if err := storageSvc.DeleteFile(ctx, filePath); err != nil {
		return err
	}

	logInfo(
		"Deleted stored file",
		zap.String("code", fileCode.Code),
		zap.String("file_path", filePath),
	)
	return nil
}

func (s *Service) storageForFile(fileCode *model.FileCode) storage.StorageInterface {
	if fileCode != nil {
		if storageType := strings.TrimSpace(fileCode.StorageType); storageType != "" {
			return storage.NewConfiguredStorageWithType(conf.GetGlobalConfig(), storageType, "")
		}
	}

	if s.storageFactory == nil {
		return nil
	}

	return s.storageFactory()
}

func (s *Service) releaseUserStorage(fileCode *model.FileCode) {
	if s.userService == nil || fileCode == nil || fileCode.UserID == nil || fileCode.Size == 0 {
		return
	}

	if err := s.userService.UpdateUserStats(*fileCode.UserID, "storage", -fileCode.Size); err != nil {
		logWarn(
			"Failed to release user storage after deleting file",
			zap.String("code", fileCode.Code),
			zap.Uint("user_id", *fileCode.UserID),
			zap.Error(err),
		)
	}
}

func logWarn(msg string, fields ...zap.Field) {
	if logger.Logger != nil {
		logger.Warn(msg, fields...)
	}
}

func logInfo(msg string, fields ...zap.Field) {
	if logger.Logger != nil {
		logger.Info(msg, fields...)
	}
}
