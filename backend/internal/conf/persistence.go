package conf

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const defaultConfigFilePath = "configs/config.yaml"

var configFilePath = defaultConfigFilePath

// SetConfigFilePath records the active config file path for later persistence.
func SetConfigFilePath(path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		configFilePath = defaultConfigFilePath
		return
	}
	configFilePath = path
}

// GetConfigFilePath returns the config file path used for persistence.
func GetConfigFilePath() string {
	if strings.TrimSpace(configFilePath) == "" {
		return defaultConfigFilePath
	}
	return configFilePath
}

// SaveGlobalConfig persists the current global config back to the active config file.
func SaveGlobalConfig() error {
	return SaveConfigToFile(globalConfig, GetConfigFilePath())
}

// SaveConfigToFile persists cfg to path while preserving unrelated YAML sections.
func SaveConfigToFile(cfg *AppConfiguration, path string) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	path = strings.TrimSpace(path)
	if path == "" {
		path = defaultConfigFilePath
	}

	rawConfig := make(map[string]interface{})
	if content, err := os.ReadFile(path); err == nil {
		if len(content) > 0 {
			if err := yaml.Unmarshal(content, &rawConfig); err != nil {
				return fmt.Errorf("解析配置文件失败: %w", err)
			}
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	serverSection := ensureMap(rawConfig, "server")
	serverSection["host"] = cfg.Server.Host
	serverSection["port"] = cfg.Server.Port
	serverSection["mode"] = cfg.Server.Mode
	serverSection["base_url"] = cfg.Server.BaseURL
	serverSection["public_base_url"] = cfg.Server.PublicBaseURL
	serverSection["read_timeout"] = cfg.Server.ReadTimeout
	serverSection["write_timeout"] = cfg.Server.WriteTimeout

	appSection := ensureMap(rawConfig, "app")
	appSection["name"] = cfg.App.Name
	appSection["version"] = cfg.App.Version
	appSection["description"] = cfg.App.Description
	appSection["datapath"] = cfg.App.DataPath
	appSection["production"] = cfg.App.Production

	userSection := ensureMap(rawConfig, "user")
	userSection["allow_user_registration"] = cfg.User.AllowUserRegistration
	userSection["require_email_verify"] = cfg.User.RequireEmailVerify
	userSection["user_upload_size"] = cfg.User.UserUploadSize
	userSection["user_storage_quota"] = cfg.User.UserStorageQuota
	userSection["session_expiry_hours"] = cfg.User.SessionExpiryHours
	userSection["max_sessions_per_user"] = cfg.User.MaxSessionsPerUser
	userSection["jwt_secret"] = cfg.User.JWTSecret

	uploadSection := ensureMap(rawConfig, "upload")
	uploadSection["open_upload"] = cfg.Upload.OpenUpload
	uploadSection["upload_size"] = cfg.Upload.UploadSize
	uploadSection["enable_chunk"] = cfg.Upload.EnableChunk
	uploadSection["chunk_size"] = cfg.Upload.ChunkSize
	uploadSection["max_save_seconds"] = cfg.Upload.MaxSaveSeconds
	uploadSection["require_login"] = cfg.Upload.RequireLogin
	uploadSection["share_code_length"] = NormalizeShareCodeLength(cfg.Upload.ShareCodeLength)
	uploadSection["share_code_charset"] = NormalizeShareCodeCharset(cfg.Upload.ShareCodeCharset)

	storageSection := ensureMap(rawConfig, "storage")
	storageSection["type"] = cfg.Storage.Type
	storageSection["storage_path"] = cfg.Storage.StoragePath

	s3Section := ensureMap(storageSection, "s3")
	s3Section["access_key_id"] = cfg.Storage.S3.AccessKeyID
	s3Section["secret_access_key"] = cfg.Storage.S3.SecretAccessKey
	s3Section["bucket_name"] = cfg.Storage.S3.BucketName
	s3Section["endpoint_url"] = cfg.Storage.S3.EndpointURL
	s3Section["region_name"] = cfg.Storage.S3.RegionName
	s3Section["hostname"] = cfg.Storage.S3.Hostname
	s3Section["proxy"] = cfg.Storage.S3.Proxy
	s3Section["signed_url_expiry"] = cfg.Storage.S3.SignedURLExpiry

	qiniuSection := ensureMap(storageSection, "qiniu")
	qiniuSection["access_key"] = cfg.Storage.Qiniu.AccessKey
	qiniuSection["secret_key"] = cfg.Storage.Qiniu.SecretKey
	qiniuSection["bucket"] = cfg.Storage.Qiniu.Bucket
	qiniuSection["domain"] = cfg.Storage.Qiniu.Domain
	qiniuSection["region"] = cfg.Storage.Qiniu.Region
	qiniuSection["use_https"] = cfg.Storage.Qiniu.UseHTTPS
	qiniuSection["private"] = cfg.Storage.Qiniu.Private

	upyunSection := ensureMap(storageSection, "upyun")
	upyunSection["bucket"] = cfg.Storage.Upyun.Bucket
	upyunSection["operator"] = cfg.Storage.Upyun.Operator
	upyunSection["password"] = cfg.Storage.Upyun.Password
	upyunSection["domain"] = cfg.Storage.Upyun.Domain
	upyunSection["secret"] = cfg.Storage.Upyun.Secret

	output, err := yaml.Marshal(rawConfig)
	if err != nil {
		return fmt.Errorf("序列化配置文件失败: %w", err)
	}

	if err := os.WriteFile(path, output, 0644); err != nil {
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
