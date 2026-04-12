package uploadpolicy

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/zy84338719/fileCodeBox/backend/internal/conf"
	"github.com/zy84338719/fileCodeBox/backend/internal/repo/db/dao"
	"github.com/zy84338719/fileCodeBox/backend/internal/repo/db/model"
)

var (
	ErrAnonymousUploadDisabled = errors.New("anonymous upload disabled")
	ErrLoginRequired           = errors.New("login required for upload")
	ErrChunkUploadDisabled     = errors.New("chunk upload disabled")
	ErrFileTooLarge            = errors.New("file too large")
	ErrStorageQuotaExceeded    = errors.New("storage quota exceeded")
)

const defaultUploadSize = 10 * 1024 * 1024

type Policy struct {
	Config        *conf.AppConfiguration
	User          *model.User
	Authenticated bool
	MaxUploadSize int64
	StorageQuota  int64
}

func Resolve(ctx context.Context, userID *uint) (*Policy, error) {
	cfg := conf.GetGlobalConfig()
	if cfg == nil {
		cfg = &conf.AppConfiguration{}
	}

	policy := &Policy{
		Config: cfg,
	}

	if userID != nil && *userID > 0 {
		user, err := dao.NewUserRepository().GetByID(ctx, *userID)
		if err != nil {
			return nil, fmt.Errorf("加载用户配置失败: %w", err)
		}

		policy.User = user
		policy.Authenticated = true
		policy.MaxUploadSize = firstPositive(user.MaxUploadSize, cfg.User.UserUploadSize, cfg.Upload.UploadSize, defaultUploadSize)
		policy.StorageQuota = firstPositive(user.MaxStorageQuota, cfg.User.UserStorageQuota)
	} else {
		policy.MaxUploadSize = firstPositive(cfg.Upload.UploadSize, defaultUploadSize)
	}

	if cfg.Upload.RequireLogin && !policy.Authenticated {
		return nil, ErrLoginRequired
	}
	if !policy.Authenticated && !cfg.Upload.OpenUpload {
		return nil, ErrAnonymousUploadDisabled
	}

	return policy, nil
}

func EnsureChunkEnabled() error {
	cfg := conf.GetGlobalConfig()
	if cfg != nil && !cfg.Upload.EnableChunk {
		return ErrChunkUploadDisabled
	}
	return nil
}

func (p *Policy) ValidateFileSize(fileSize int64) error {
	if fileSize <= 0 || p == nil || p.MaxUploadSize <= 0 {
		return nil
	}
	if fileSize > p.MaxUploadSize {
		return fmt.Errorf("%w: 最大允许 %s", ErrFileTooLarge, FormatSize(p.MaxUploadSize))
	}
	return nil
}

func (p *Policy) ValidateStorageQuota(fileSize int64) error {
	if p == nil || !p.Authenticated || p.User == nil || p.StorageQuota <= 0 || fileSize <= 0 {
		return nil
	}
	if p.User.TotalStorage+fileSize > p.StorageQuota {
		return fmt.Errorf("%w: 当前配额 %s", ErrStorageQuotaExceeded, FormatSize(p.StorageQuota))
	}
	return nil
}

func FormatSize(size int64) string {
	if size <= 0 {
		return "0 B"
	}

	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	value := float64(size)
	unitIndex := 0
	for value >= 1024 && unitIndex < len(units)-1 {
		value /= 1024
		unitIndex++
	}

	rounded := math.Round(value*100) / 100
	formatted := fmt.Sprintf("%.2f", rounded)
	formatted = strings.TrimRight(strings.TrimRight(formatted, "0"), ".")
	return fmt.Sprintf("%s %s", formatted, units[unitIndex])
}

func firstPositive(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
