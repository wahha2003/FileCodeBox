package conf

import (
	"fmt"
	"strings"
)

const (
	DefaultShareCodeLength  = 4
	MaxShareCodeLength      = 32
	DefaultShareCodeCharset = "0123456789"
	shareCodeAllowedCharset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

// 全局配置
var globalConfig *AppConfiguration

// AppConfiguration 完整应用配置
type AppConfiguration struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Log      LogConfig      `mapstructure:"log"`
	App      AppConfig      `mapstructure:"app"`
	User     UserConfig     `mapstructure:"user"`
	Upload   UploadConfig   `mapstructure:"upload"`
	Download DownloadConfig `mapstructure:"download"`
	Storage  StorageConfig  `mapstructure:"storage"`
}

// SetGlobalConfig 设置全局配置
func SetGlobalConfig(cfg *AppConfiguration) {
	globalConfig = cfg
}

// GetGlobalConfig 获取全局配置
func GetGlobalConfig() *AppConfiguration {
	return globalConfig
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host          string `mapstructure:"host"`
	Port          int    `mapstructure:"port"`
	Mode          string `mapstructure:"mode"` // debug, release, test
	BaseURL       string `mapstructure:"base_url"`
	PublicBaseURL string `mapstructure:"public_base_url"`
	ReadTimeout   int    `mapstructure:"read_timeout"`
	WriteTimeout  int    `mapstructure:"write_timeout"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver   string `mapstructure:"driver"` // sqlite, mysql, postgres
	DBName   string `mapstructure:"db_name"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// Addr 返回 Redis 地址
func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

// AppConfig 应用配置
type AppConfig struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	Description string `mapstructure:"description"`
	DataPath    string `mapstructure:"datapath"`
	Production  bool   `mapstructure:"production"`
}

// UserConfig 用户配置
type UserConfig struct {
	AllowUserRegistration bool   `mapstructure:"allow_user_registration"`
	RequireEmailVerify    bool   `mapstructure:"require_email_verify"`
	UserUploadSize        int64  `mapstructure:"user_upload_size"`
	UserStorageQuota      int64  `mapstructure:"user_storage_quota"`
	SessionExpiryHours    int    `mapstructure:"session_expiry_hours"`
	MaxSessionsPerUser    int    `mapstructure:"max_sessions_per_user"`
	JWTSecret             string `mapstructure:"jwt_secret"`
}

// UploadConfig 上传配置
type UploadConfig struct {
	OpenUpload       bool   `mapstructure:"open_upload"`
	UploadSize       int64  `mapstructure:"upload_size"`
	EnableChunk      bool   `mapstructure:"enable_chunk"`
	ChunkSize        int64  `mapstructure:"chunk_size"`
	MaxSaveSeconds   int    `mapstructure:"max_save_seconds"`
	RequireLogin     bool   `mapstructure:"require_login"`
	ShareCodeLength  int    `mapstructure:"share_code_length"`
	ShareCodeCharset string `mapstructure:"share_code_charset"`
}

// DownloadConfig 下载配置
type DownloadConfig struct {
	EnableConcurrentDownload bool `mapstructure:"enable_concurrent_download"`
	MaxConcurrentDownloads   int  `mapstructure:"max_concurrent_downloads"`
	DownloadTimeout          int  `mapstructure:"download_timeout"`
	RequireLogin             bool `mapstructure:"require_login"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Type        string      `mapstructure:"type"`
	StoragePath string      `mapstructure:"storage_path"`
	S3          S3Config    `mapstructure:"s3"`
	Qiniu       QiniuConfig `mapstructure:"qiniu"`
	Upyun       UpyunConfig `mapstructure:"upyun"`
}

type S3Config struct {
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey  string `mapstructure:"secret_access_key"`
	BucketName       string `mapstructure:"bucket_name"`
	EndpointURL      string `mapstructure:"endpoint_url"`
	RegionName       string `mapstructure:"region_name"`
	Hostname         string `mapstructure:"hostname"`
	Proxy            string `mapstructure:"proxy"`
	SignedURLExpiry  int    `mapstructure:"signed_url_expiry"` // 预签名 URL 有效期（秒），默认 3600
}

// QiniuConfig 七牛云配置
type QiniuConfig struct {
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	Bucket    string `mapstructure:"bucket"`
	Domain    string `mapstructure:"domain"`    // CDN 域名
	Region    string `mapstructure:"region"`    // 区域: z0/z1/z2/na0/as0/cn-east-2
	UseHTTPS  bool   `mapstructure:"use_https"`
	Private   bool   `mapstructure:"private"`   // 是否私有空间
}

// UpyunConfig 又拍云配置
type UpyunConfig struct {
	Bucket   string `mapstructure:"bucket"`   // 服务名
	Operator string `mapstructure:"operator"` // 操作员
	Password string `mapstructure:"password"` // 操作密码
	Domain   string `mapstructure:"domain"`   // CDN 域名
	Secret   string `mapstructure:"secret"`   // Token 防盗链密钥
}

func NormalizeShareCodeLength(length int) int {
	switch {
	case length <= 0:
		return DefaultShareCodeLength
	case length > MaxShareCodeLength:
		return MaxShareCodeLength
	default:
		return length
	}
}

func NormalizeShareCodeCharset(charset string) string {
	charset = strings.TrimSpace(charset)
	if charset == "" {
		return DefaultShareCodeCharset
	}

	allowed := make(map[rune]struct{}, len(shareCodeAllowedCharset))
	for _, ch := range shareCodeAllowedCharset {
		allowed[ch] = struct{}{}
	}

	seen := make(map[rune]struct{}, len(charset))
	var builder strings.Builder
	for _, ch := range charset {
		if _, ok := allowed[ch]; !ok {
			continue
		}
		if _, ok := seen[ch]; ok {
			continue
		}
		seen[ch] = struct{}{}
		builder.WriteRune(ch)
	}

	if builder.Len() == 0 {
		return DefaultShareCodeCharset
	}

	return builder.String()
}

func GetShareCodeConfig(cfg *AppConfiguration) (int, string) {
	if cfg == nil {
		return DefaultShareCodeLength, DefaultShareCodeCharset
	}

	return NormalizeShareCodeLength(cfg.Upload.ShareCodeLength), NormalizeShareCodeCharset(cfg.Upload.ShareCodeCharset)
}

func IsNumericShareCodeCharset(charset string) bool {
	charset = NormalizeShareCodeCharset(charset)
	for _, ch := range charset {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return charset != ""
}
