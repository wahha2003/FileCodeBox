package storage

import (
	"strings"

	"github.com/zy84338719/fileCodeBox/backend/internal/conf"
)

const (
	defaultLocalDataPath = "./data"
)

// BuildStorageConfigFromAppConfig 将应用配置转换为存储配置。
func BuildStorageConfigFromAppConfig(cfg *conf.AppConfiguration, fallbackBaseURL string) *StorageConfig {
	storageCfg := &StorageConfig{
		Type:     StorageTypeLocal,
		DataPath: defaultLocalDataPath,
		BaseURL:  fallbackBaseURL,
	}

	if cfg == nil {
		return storageCfg
	}

	if strings.TrimSpace(cfg.Server.BaseURL) != "" {
		storageCfg.BaseURL = strings.TrimSpace(cfg.Server.BaseURL)
	}
	if strings.TrimSpace(cfg.App.DataPath) != "" {
		storageCfg.DataPath = strings.TrimSpace(cfg.App.DataPath)
	}
	if strings.TrimSpace(cfg.Storage.StoragePath) != "" {
		storageCfg.DataPath = strings.TrimSpace(cfg.Storage.StoragePath)
	}
	if strings.TrimSpace(cfg.Storage.Type) != "" {
		storageCfg.Type = StorageType(strings.ToLower(strings.TrimSpace(cfg.Storage.Type)))
	}

	// S3 配置
	storageCfg.AccessKey = strings.TrimSpace(cfg.Storage.S3.AccessKeyID)
	storageCfg.SecretKey = strings.TrimSpace(cfg.Storage.S3.SecretAccessKey)
	storageCfg.Bucket = strings.TrimSpace(cfg.Storage.S3.BucketName)
	storageCfg.Endpoint = strings.TrimSpace(cfg.Storage.S3.EndpointURL)
	storageCfg.Region = strings.TrimSpace(cfg.Storage.S3.RegionName)
	storageCfg.Hostname = strings.TrimSpace(cfg.Storage.S3.Hostname)
	storageCfg.Proxy = strings.TrimSpace(cfg.Storage.S3.Proxy)
	storageCfg.SignedURLExpiry = cfg.Storage.S3.SignedURLExpiry

	// 七牛云配置
	storageCfg.QiniuAccessKey = strings.TrimSpace(cfg.Storage.Qiniu.AccessKey)
	storageCfg.QiniuSecretKey = strings.TrimSpace(cfg.Storage.Qiniu.SecretKey)
	storageCfg.QiniuBucket = strings.TrimSpace(cfg.Storage.Qiniu.Bucket)
	storageCfg.QiniuDomain = strings.TrimSpace(cfg.Storage.Qiniu.Domain)
	storageCfg.QiniuRegion = strings.TrimSpace(cfg.Storage.Qiniu.Region)
	storageCfg.QiniuUseHTTPS = cfg.Storage.Qiniu.UseHTTPS
	storageCfg.QiniuPrivate = cfg.Storage.Qiniu.Private

	// 又拍云配置
	storageCfg.UpyunBucket = strings.TrimSpace(cfg.Storage.Upyun.Bucket)
	storageCfg.UpyunOperator = strings.TrimSpace(cfg.Storage.Upyun.Operator)
	storageCfg.UpyunPassword = strings.TrimSpace(cfg.Storage.Upyun.Password)
	storageCfg.UpyunDomain = strings.TrimSpace(cfg.Storage.Upyun.Domain)
	storageCfg.UpyunSecret = strings.TrimSpace(cfg.Storage.Upyun.Secret)

	return storageCfg
}

// NewConfiguredStorage 根据应用配置创建实际存储实例。
func NewConfiguredStorage(cfg *conf.AppConfiguration, fallbackBaseURL string) StorageInterface {
	storageCfg := BuildStorageConfigFromAppConfig(cfg, fallbackBaseURL)

	switch storageCfg.Type {
	case StorageTypeS3:
		return NewS3Storage(storageCfg)
	case StorageTypeQiniu:
		return NewQiniuStorage(storageCfg)
	case StorageTypeUpyun:
		return NewUpyunStorage(storageCfg)
	default:
		return NewStorageService(storageCfg)
	}
}

