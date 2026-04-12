package storage

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	storage "github.com/zy84338719/fileCodeBox/backend/gen/http/model/storage"
	storagesvc "github.com/zy84338719/fileCodeBox/backend/internal/app/storage"
)

var storageService *storagesvc.Service

func init() {
	storageService = storagesvc.NewService()
}

// GetStorageInfo .
// @router /admin/storage [GET]
func GetStorageInfo(ctx context.Context, c *app.RequestContext) {
	var req storage.GetStorageInfoReq
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, &storage.GetStorageInfoResp{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	info, err := storageService.GetStorageInfo(ctx)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, &storage.GetStorageInfoResp{
			Code:    500,
			Message: "获取存储信息失败: " + err.Error(),
		})
		return
	}

	// NOTE: The generated proto model does not include qiniu/upyun fields yet.
	// Return the service-layer payload directly so the admin UI can save and refill
	// all storage configs consistently.
	c.JSON(consts.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "获取成功",
		"data":    info,
	})
}

// SwitchStorage .
// @router /admin/storage/switch [POST]
func SwitchStorage(ctx context.Context, c *app.RequestContext) {
	var req storage.SwitchStorageReq
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, &storage.SwitchStorageResp{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	if req.Type == "" {
		c.JSON(consts.StatusBadRequest, &storage.SwitchStorageResp{
			Code:    400,
			Message: "存储类型不能为空",
		})
		return
	}

	if err := storageService.SwitchStorage(ctx, req.Type); err != nil {
		c.JSON(consts.StatusInternalServerError, &storage.SwitchStorageResp{
			Code:    500,
			Message: "存储切换失败: " + err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, &storage.SwitchStorageResp{
		Code:    200,
		Message: "切换成功",
		Data: &storage.SwitchStorageData{
			Success:     true,
			Message:     "存储切换成功",
			CurrentType: req.Type,
		},
	})
}

// TestStorageConnection .
// @router /admin/storage/test/:type [GET]
func TestStorageConnection(ctx context.Context, c *app.RequestContext) {
	var req storage.TestStorageConnectionReq
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, &storage.TestStorageConnectionResp{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	if err := storageService.TestStorageConnection(ctx, req.Type); err != nil {
		c.JSON(consts.StatusBadRequest, &storage.TestStorageConnectionResp{
			Code:    400,
			Message: "连接测试失败: " + err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, &storage.TestStorageConnectionResp{
		Code:    200,
		Message: "连接测试成功",
		Data: &storage.TestStorageData{
			Type:   req.Type,
			Status: "connected",
		},
	})
}

// UpdateStorageConfig .
// @router /admin/storage/config [PUT]
func UpdateStorageConfig(ctx context.Context, c *app.RequestContext) {
	var req updateStorageConfigPayload
	if err := c.Bind(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]interface{}{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	if req.Type == "" {
		c.JSON(consts.StatusBadRequest, map[string]interface{}{
			"code":    400,
			"message": "存储类型不能为空",
		})
		return
	}

	updateReq := &storagesvc.UpdateConfigRequest{
		Type: req.Type,
	}
	if req.Config != nil {
		updateReq.Config.StoragePath = req.Config.StoragePath
		updateReq.Config.WebDAV = req.Config.WebDAV
		updateReq.Config.S3 = req.Config.S3
		updateReq.Config.Qiniu = req.Config.Qiniu
		updateReq.Config.Upyun = req.Config.Upyun
		updateReq.Config.NFS = req.Config.NFS
	}

	if err := storageService.UpdateStorageConfig(ctx, updateReq); err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]interface{}{
			"code":    500,
			"message": "配置更新失败: " + err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "存储配置更新成功",
	})
}

type updateStorageConfigPayload struct {
	Type   string                   `json:"type"`
	Config *updateStorageConfigBody `json:"config"`
}

type updateStorageConfigBody struct {
	StoragePath string                   `json:"storage_path"`
	WebDAV      *storagesvc.WebDAVConfig `json:"webdav"`
	S3          *storagesvc.S3Config     `json:"s3"`
	Qiniu       *storagesvc.QiniuConfig  `json:"qiniu"`
	Upyun       *storagesvc.UpyunConfig  `json:"upyun"`
	NFS         *storagesvc.NFSConfig    `json:"nfs"`
}

func convertToProtoStorageConfig(cfg *storagesvc.StorageConfig) *storage.StorageConfig {
	if cfg == nil {
		return nil
	}
	return &storage.StorageConfig{
		Type:        cfg.Type,
		StoragePath: cfg.StoragePath,
		Webdav:      convertToProtoWebDAVConfig(cfg.WebDAV),
		S3:          convertToProtoS3Config(cfg.S3),
		Nfs:         convertToProtoNFSConfig(cfg.NFS),
	}
}

func convertToProtoWebDAVConfig(cfg *storagesvc.WebDAVConfig) *storage.WebDAVConfig {
	if cfg == nil {
		return nil
	}
	return &storage.WebDAVConfig{
		Hostname: cfg.Hostname,
		Username: cfg.Username,
		Password: cfg.Password,
		RootPath: cfg.RootPath,
		Url:      cfg.URL,
	}
}

func convertToProtoS3Config(cfg *storagesvc.S3Config) *storage.S3Config {
	if cfg == nil {
		return nil
	}
	return &storage.S3Config{
		AccessKeyId:     cfg.AccessKeyID,
		SecretAccessKey: cfg.SecretAccessKey,
		BucketName:      cfg.BucketName,
		EndpointUrl:     cfg.EndpointURL,
		RegionName:      cfg.RegionName,
		Hostname:        cfg.Hostname,
		Proxy:           cfg.Proxy,
	}
}

func convertToProtoNFSConfig(cfg *storagesvc.NFSConfig) *storage.NFSConfig {
	if cfg == nil {
		return nil
	}
	return &storage.NFSConfig{
		Server:     cfg.Server,
		Path:       cfg.Path,
		MountPoint: cfg.MountPoint,
		Version:    cfg.Version,
		Options:    cfg.Options,
		Timeout:    cfg.Timeout,
		AutoMount:  cfg.AutoMount,
		RetryCount: cfg.RetryCount,
		SubPath:    cfg.SubPath,
	}
}

func convertFromProtoWebDAVConfig(cfg *storage.WebDAVConfig) *storagesvc.WebDAVConfig {
	if cfg == nil {
		return nil
	}
	return &storagesvc.WebDAVConfig{
		Hostname: cfg.Hostname,
		Username: cfg.Username,
		Password: cfg.Password,
		RootPath: cfg.RootPath,
		URL:      cfg.Url,
	}
}

func convertFromProtoS3Config(cfg *storage.S3Config) *storagesvc.S3Config {
	if cfg == nil {
		return nil
	}
	return &storagesvc.S3Config{
		AccessKeyID:     cfg.AccessKeyId,
		SecretAccessKey: cfg.SecretAccessKey,
		BucketName:      cfg.BucketName,
		EndpointURL:     cfg.EndpointUrl,
		RegionName:      cfg.RegionName,
		Hostname:        cfg.Hostname,
		Proxy:           cfg.Proxy,
	}
}

func convertFromProtoNFSConfig(cfg *storage.NFSConfig) *storagesvc.NFSConfig {
	if cfg == nil {
		return nil
	}
	return &storagesvc.NFSConfig{
		Server:     cfg.Server,
		Path:       cfg.Path,
		MountPoint: cfg.MountPoint,
		Version:    cfg.Version,
		Options:    cfg.Options,
		Timeout:    cfg.Timeout,
		AutoMount:  cfg.AutoMount,
		RetryCount: cfg.RetryCount,
		SubPath:    cfg.SubPath,
	}
}
