package storage

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/zy84338719/fileCodeBox/backend/internal/conf"
	storageimpl "github.com/zy84338719/fileCodeBox/backend/internal/storage"
	"gopkg.in/yaml.v3"
)

// Service 存储服务
type Service struct {
	config *conf.AppConfiguration
}

// NewService 创建存储服务
func NewService() *Service {
	s := &Service{}
	s.refreshConfig()
	return s
}

// GetStorageInfo 获取存储信息
func (s *Service) GetStorageInfo(ctx context.Context) (*StorageInfo, error) {
	s.refreshConfig()

	storageDetails := map[string]*StorageDetail{
		"local": {
			Type:         "local",
			Available:    true,
			StoragePath:  s.getStoragePath(),
			UsagePercent: s.getDiskUsage(),
		},
		"s3": {
			Type:         "s3",
			Available:    s.hasS3Config(),
			StoragePath:  s.describeS3Target(),
			UsagePercent: 0,
		},
		"qiniu": {
			Type:         "qiniu",
			Available:    s.hasQiniuConfig(),
			StoragePath:  s.describeQiniuTarget(),
			UsagePercent: 0,
		},
		"upyun": {
			Type:         "upyun",
			Available:    s.hasUpyunConfig(),
			StoragePath:  s.describeUpyunTarget(),
			UsagePercent: 0,
		},
	}

	if !s.hasS3Config() {
		storageDetails["s3"].Error = "S3 配置未完成"
	}
	if !s.hasQiniuConfig() {
		storageDetails["qiniu"].Error = "七牛云配置未完成"
	}
	if !s.hasUpyunConfig() {
		storageDetails["upyun"].Error = "又拍云配置未完成"
	}

	return &StorageInfo{
		Current:        s.currentType(),
		Available:      []string{"local", "s3", "qiniu", "upyun"},
		StorageDetails: storageDetails,
		StorageConfig:  s.getStorageConfig(),
	}, nil
}

// SwitchStorage 切换存储类型
func (s *Service) SwitchStorage(ctx context.Context, storageType string) error {
	s.refreshConfig()

	targetType, err := normalizeStorageType(storageType)
	if err != nil {
		return err
	}

	if targetType == "s3" || targetType == "qiniu" || targetType == "upyun" {
		if err := s.TestStorageConnection(ctx, targetType); err != nil {
			return err
		}
	}

	s.config.Storage.Type = targetType
	conf.SetGlobalConfig(s.config)

	return s.persistStorageConfig()
}

// TestStorageConnection 测试存储连接
func (s *Service) TestStorageConnection(ctx context.Context, storageType string) error {
	s.refreshConfig()

	targetType, err := normalizeStorageType(storageType)
	if err != nil {
		return err
	}

	switch targetType {
	case "local":
		path := s.getStoragePath()
		if path == "" {
			return fmt.Errorf("存储路径未配置")
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("存储路径不存在: %s", path)
		}

		testFile := path + "/.test_write"
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			return fmt.Errorf("存储路径不可写: %w", err)
		}
		_ = os.Remove(testFile)
		return nil

	case "s3", "qiniu", "upyun":
		runtimeStorage := s.buildRuntimeStorage(targetType)
		tester, ok := runtimeStorage.(storageimpl.ConnectionTester)
		if !ok {
			return fmt.Errorf("当前存储实现不支持连接测试")
		}
		return tester.TestConnection(ctx)

	default:
		return fmt.Errorf("不支持的存储类型: %s", targetType)
	}
}

// UpdateStorageConfig 更新存储配置
func (s *Service) UpdateStorageConfig(ctx context.Context, req *UpdateConfigRequest) error {
	_ = ctx
	s.refreshConfig()

	targetType, err := normalizeStorageType(req.Type)
	if err != nil {
		return err
	}

	switch targetType {
	case "local":
		if strings.TrimSpace(req.Config.StoragePath) == "" {
			return fmt.Errorf("本地存储路径不能为空")
		}
		s.config.Storage.StoragePath = strings.TrimSpace(req.Config.StoragePath)

	case "s3":
		if req.Config.S3 == nil {
			return fmt.Errorf("S3 配置不能为空")
		}
		s.config.Storage.S3.AccessKeyID = strings.TrimSpace(req.Config.S3.AccessKeyID)
		s.config.Storage.S3.SecretAccessKey = strings.TrimSpace(req.Config.S3.SecretAccessKey)
		s.config.Storage.S3.BucketName = strings.TrimSpace(req.Config.S3.BucketName)
		s.config.Storage.S3.EndpointURL = strings.TrimSpace(req.Config.S3.EndpointURL)
		s.config.Storage.S3.RegionName = strings.TrimSpace(req.Config.S3.RegionName)
		s.config.Storage.S3.Hostname = strings.TrimSpace(req.Config.S3.Hostname)
		s.config.Storage.S3.Proxy = strings.TrimSpace(req.Config.S3.Proxy)

	case "qiniu":
		if req.Config.Qiniu == nil {
			return fmt.Errorf("七牛云配置不能为空")
		}
		s.config.Storage.Qiniu.AccessKey = strings.TrimSpace(req.Config.Qiniu.AccessKey)
		s.config.Storage.Qiniu.SecretKey = strings.TrimSpace(req.Config.Qiniu.SecretKey)
		s.config.Storage.Qiniu.Bucket = strings.TrimSpace(req.Config.Qiniu.Bucket)
		s.config.Storage.Qiniu.Domain = strings.TrimSpace(req.Config.Qiniu.Domain)
		s.config.Storage.Qiniu.Region = strings.TrimSpace(req.Config.Qiniu.Region)
		s.config.Storage.Qiniu.UseHTTPS = req.Config.Qiniu.UseHTTPS
		s.config.Storage.Qiniu.Private = req.Config.Qiniu.Private

	case "upyun":
		if req.Config.Upyun == nil {
			return fmt.Errorf("又拍云配置不能为空")
		}
		s.config.Storage.Upyun.Bucket = strings.TrimSpace(req.Config.Upyun.Bucket)
		s.config.Storage.Upyun.Operator = strings.TrimSpace(req.Config.Upyun.Operator)
		s.config.Storage.Upyun.Password = strings.TrimSpace(req.Config.Upyun.Password)
		s.config.Storage.Upyun.Domain = strings.TrimSpace(req.Config.Upyun.Domain)
		s.config.Storage.Upyun.Secret = strings.TrimSpace(req.Config.Upyun.Secret)

	default:
		return fmt.Errorf("不支持的存储类型: %s", targetType)
	}

	conf.SetGlobalConfig(s.config)
	return s.persistStorageConfig()
}

// getStoragePath 获取存储路径
func (s *Service) getStoragePath() string {
	if s.config == nil {
		return "./data"
	}
	if s.config.Storage.StoragePath != "" {
		return s.config.Storage.StoragePath
	}
	if s.config.App.DataPath != "" {
		return s.config.App.DataPath
	}
	return "./data"
}

// getDiskUsage 获取磁盘使用率
func (s *Service) getDiskUsage() int32 {
	return 0
}

// getStorageConfig 获取存储配置
func (s *Service) getStorageConfig() *StorageConfig {
	return &StorageConfig{
		Type:        s.currentType(),
		StoragePath: s.getStoragePath(),
		WebDAV:      s.getWebDAVConfig(),
		S3:          s.getS3Config(),
		Qiniu:       s.getQiniuConfig(),
		Upyun:       s.getUpyunConfig(),
		NFS:         s.getNFSConfig(),
	}
}

// getWebDAVConfig 获取 WebDAV 配置
func (s *Service) getWebDAVConfig() *WebDAVConfig {
	return &WebDAVConfig{}
}

// getS3Config 获取 S3 配置
func (s *Service) getS3Config() *S3Config {
	return &S3Config{
		AccessKeyID:     s.config.Storage.S3.AccessKeyID,
		SecretAccessKey: s.config.Storage.S3.SecretAccessKey,
		BucketName:      s.config.Storage.S3.BucketName,
		EndpointURL:     s.config.Storage.S3.EndpointURL,
		RegionName:      s.config.Storage.S3.RegionName,
		Hostname:        s.config.Storage.S3.Hostname,
		Proxy:           s.config.Storage.S3.Proxy,
	}
}

// getNFSConfig 获取 NFS 配置
func (s *Service) getNFSConfig() *NFSConfig {
	return &NFSConfig{}
}

func (s *Service) refreshConfig() {
	s.config = conf.GetGlobalConfig()
	if s.config == nil {
		s.config = &conf.AppConfiguration{}
		conf.SetGlobalConfig(s.config)
	}
}

func (s *Service) currentType() string {
	if s.config == nil || strings.TrimSpace(s.config.Storage.Type) == "" {
		return "local"
	}
	return strings.ToLower(strings.TrimSpace(s.config.Storage.Type))
}

func (s *Service) hasS3Config() bool {
	if s.config == nil {
		return false
	}
	return strings.TrimSpace(s.config.Storage.S3.AccessKeyID) != "" &&
		strings.TrimSpace(s.config.Storage.S3.SecretAccessKey) != "" &&
		strings.TrimSpace(s.config.Storage.S3.BucketName) != ""
}

func (s *Service) describeS3Target() string {
	if s.config == nil {
		return "-"
	}

	target := strings.TrimSpace(s.config.Storage.S3.EndpointURL)
	if target == "" {
		target = strings.TrimSpace(s.config.Storage.S3.Hostname)
	}
	if target == "" {
		target = "AWS S3"
	}

	bucket := strings.TrimSpace(s.config.Storage.S3.BucketName)
	if bucket == "" {
		return target
	}
	return fmt.Sprintf("%s / %s", bucket, target)
}

func (s *Service) hasQiniuConfig() bool {
	if s.config == nil {
		return false
	}
	return strings.TrimSpace(s.config.Storage.Qiniu.AccessKey) != "" &&
		strings.TrimSpace(s.config.Storage.Qiniu.SecretKey) != "" &&
		strings.TrimSpace(s.config.Storage.Qiniu.Bucket) != ""
}

func (s *Service) describeQiniuTarget() string {
	if s.config == nil {
		return "-"
	}
	bucket := strings.TrimSpace(s.config.Storage.Qiniu.Bucket)
	domain := strings.TrimSpace(s.config.Storage.Qiniu.Domain)
	if bucket == "" {
		return "-"
	}
	if domain != "" {
		return fmt.Sprintf("%s / %s", bucket, domain)
	}
	return bucket
}

func (s *Service) hasUpyunConfig() bool {
	if s.config == nil {
		return false
	}
	return strings.TrimSpace(s.config.Storage.Upyun.Bucket) != "" &&
		strings.TrimSpace(s.config.Storage.Upyun.Operator) != "" &&
		strings.TrimSpace(s.config.Storage.Upyun.Password) != ""
}

func (s *Service) describeUpyunTarget() string {
	if s.config == nil {
		return "-"
	}
	bucket := strings.TrimSpace(s.config.Storage.Upyun.Bucket)
	domain := strings.TrimSpace(s.config.Storage.Upyun.Domain)
	if bucket == "" {
		return "-"
	}
	if domain != "" {
		return fmt.Sprintf("%s / %s", bucket, domain)
	}
	return bucket
}

func (s *Service) getQiniuConfig() *QiniuConfig {
	return &QiniuConfig{
		AccessKey: s.config.Storage.Qiniu.AccessKey,
		SecretKey: s.config.Storage.Qiniu.SecretKey,
		Bucket:    s.config.Storage.Qiniu.Bucket,
		Domain:    s.config.Storage.Qiniu.Domain,
		Region:    s.config.Storage.Qiniu.Region,
		UseHTTPS:  s.config.Storage.Qiniu.UseHTTPS,
		Private:   s.config.Storage.Qiniu.Private,
	}
}

func (s *Service) getUpyunConfig() *UpyunConfig {
	return &UpyunConfig{
		Bucket:   s.config.Storage.Upyun.Bucket,
		Operator: s.config.Storage.Upyun.Operator,
		Password: s.config.Storage.Upyun.Password,
		Domain:   s.config.Storage.Upyun.Domain,
		Secret:   s.config.Storage.Upyun.Secret,
	}
}

func (s *Service) persistStorageConfig() error {
	s.refreshConfig()

	rawConfig := make(map[string]interface{})
	configFilePath := conf.GetConfigFilePath()
	if content, err := os.ReadFile(configFilePath); err == nil {
		if len(content) > 0 {
			if err := yaml.Unmarshal(content, &rawConfig); err != nil {
				return fmt.Errorf("解析配置文件失败: %w", err)
			}
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	storageSection := ensureMap(rawConfig, "storage")
	storageSection["type"] = s.currentType()
	storageSection["storage_path"] = s.getStoragePath()
	storageSection["s3"] = map[string]interface{}{
		"access_key_id":     s.config.Storage.S3.AccessKeyID,
		"secret_access_key": s.config.Storage.S3.SecretAccessKey,
		"bucket_name":       s.config.Storage.S3.BucketName,
		"endpoint_url":      s.config.Storage.S3.EndpointURL,
		"region_name":       s.config.Storage.S3.RegionName,
		"hostname":          s.config.Storage.S3.Hostname,
		"proxy":             s.config.Storage.S3.Proxy,
		"signed_url_expiry": s.config.Storage.S3.SignedURLExpiry,
	}
	storageSection["qiniu"] = map[string]interface{}{
		"access_key": s.config.Storage.Qiniu.AccessKey,
		"secret_key": s.config.Storage.Qiniu.SecretKey,
		"bucket":     s.config.Storage.Qiniu.Bucket,
		"domain":     s.config.Storage.Qiniu.Domain,
		"region":     s.config.Storage.Qiniu.Region,
		"use_https":  s.config.Storage.Qiniu.UseHTTPS,
		"private":    s.config.Storage.Qiniu.Private,
	}
	storageSection["upyun"] = map[string]interface{}{
		"bucket":   s.config.Storage.Upyun.Bucket,
		"operator": s.config.Storage.Upyun.Operator,
		"password": s.config.Storage.Upyun.Password,
		"domain":   s.config.Storage.Upyun.Domain,
		"secret":   s.config.Storage.Upyun.Secret,
	}

	output, err := yaml.Marshal(rawConfig)
	if err != nil {
		return fmt.Errorf("序列化配置文件失败: %w", err)
	}

	if err := os.WriteFile(configFilePath, output, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

func ensureMap(target map[string]interface{}, key string) map[string]interface{} {
	if existing, ok := target[key].(map[string]interface{}); ok {
		return existing
	}

	newMap := make(map[string]interface{})
	target[key] = newMap
	return newMap
}

// ==================== 响应模型 ====================

type StorageInfo struct {
	Current        string                    `json:"current"`
	Available      []string                  `json:"available"`
	StorageDetails map[string]*StorageDetail `json:"storage_details"`
	StorageConfig  *StorageConfig            `json:"storage_config"`
}

type StorageDetail struct {
	Type         string `json:"type"`
	Available    bool   `json:"available"`
	StoragePath  string `json:"storage_path"`
	UsagePercent int32  `json:"usage_percent"`
	Error        string `json:"error,omitempty"`
}

type StorageConfig struct {
	Type        string        `json:"type"`
	StoragePath string        `json:"storage_path"`
	WebDAV      *WebDAVConfig `json:"webdav,omitempty"`
	S3          *S3Config     `json:"s3,omitempty"`
	Qiniu       *QiniuConfig  `json:"qiniu,omitempty"`
	Upyun       *UpyunConfig  `json:"upyun,omitempty"`
	NFS         *NFSConfig    `json:"nfs,omitempty"`
}

type WebDAVConfig struct {
	Hostname string `json:"hostname"`
	Username string `json:"username"`
	Password string `json:"password"`
	RootPath string `json:"root_path"`
	URL      string `json:"url"`
}

type S3Config struct {
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	BucketName      string `json:"bucket_name"`
	EndpointURL     string `json:"endpoint_url"`
	RegionName      string `json:"region_name"`
	Hostname        string `json:"hostname"`
	Proxy           string `json:"proxy"`
}

// QiniuConfig 七牛云配置
type QiniuConfig struct {
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Bucket    string `json:"bucket"`
	Domain    string `json:"domain"`
	Region    string `json:"region"`
	UseHTTPS  bool   `json:"use_https"`
	Private   bool   `json:"private"`
}

// UpyunConfig 又拍云配置
type UpyunConfig struct {
	Bucket   string `json:"bucket"`
	Operator string `json:"operator"`
	Password string `json:"password"`
	Domain   string `json:"domain"`
	Secret   string `json:"secret"`
}

type NFSConfig struct {
	Server     string `json:"server"`
	Path       string `json:"path"`
	MountPoint string `json:"mount_point"`
	Version    string `json:"version"`
	Options    string `json:"options"`
	Timeout    int32  `json:"timeout"`
	AutoMount  int32  `json:"auto_mount"`
	RetryCount int32  `json:"retry_count"`
	SubPath    string `json:"sub_path"`
}

// UpdateConfigRequest 更新配置请求
type UpdateConfigRequest struct {
	Type   string `json:"type"`
	Config struct {
		StoragePath string        `json:"storage_path"`
		WebDAV      *WebDAVConfig `json:"webdav"`
		S3          *S3Config     `json:"s3"`
		Qiniu       *QiniuConfig  `json:"qiniu"`
		Upyun       *UpyunConfig  `json:"upyun"`
		NFS         *NFSConfig    `json:"nfs"`
	} `json:"config"`
}

func (s *Service) buildRuntimeStorage(storageType string) storageimpl.StorageInterface {
	cfgCopy := *s.config
	cfgCopy.Storage = s.config.Storage
	cfgCopy.Storage.Type = storageType
	return storageimpl.NewConfiguredStorage(&cfgCopy, "")
}

func normalizeStorageType(storageType string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(storageType)) {
	case "local":
		return "local", nil
	case "s3":
		return "s3", nil
	case "qiniu":
		return "qiniu", nil
	case "upyun":
		return "upyun", nil
	default:
		return "", fmt.Errorf("不支持的存储类型: %s", storageType)
	}
}
