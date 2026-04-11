package admin

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	admin "github.com/zy84338719/fileCodeBox/backend/gen/http/model/admin"
	adminsvc "github.com/zy84338719/fileCodeBox/backend/internal/app/admin"
	"github.com/zy84338719/fileCodeBox/backend/internal/conf"
	"github.com/zy84338719/fileCodeBox/backend/internal/repo/db/dao"
	dbmodel "github.com/zy84338719/fileCodeBox/backend/internal/repo/db/model"
	"github.com/zy84338719/fileCodeBox/backend/internal/storage"
)

var adminService *adminsvc.Service

func init() {
	adminService = adminsvc.NewService()
}

// AdminLogin .
// @router /admin/login [POST]
func AdminLogin(ctx context.Context, c *app.RequestContext) {
	var req admin.AdminLoginReq
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]interface{}{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	if req.Username == "" || req.Password == "" {
		c.JSON(consts.StatusBadRequest, map[string]interface{}{
			"code":    400,
			"message": "用户名和密码不能为空",
		})
		return
	}

	token, err := adminService.GenerateTokenForAdmin(ctx, req.Username, req.Password)
	if err != nil {
		c.JSON(consts.StatusUnauthorized, map[string]interface{}{
			"code":    401,
			"message": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "登录成功",
		"data": map[string]interface{}{
			"token":      token,
			"token_type": "Bearer",
			"expires_in": 24 * 60 * 60,
			"user": map[string]interface{}{
				"username": req.Username,
				"role":     "admin",
			},
		},
	})
}

// AdminStats .
// @router /admin/stats [GET]
func AdminStats(ctx context.Context, c *app.RequestContext) {
	stats, err := adminService.GetStats(ctx)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]interface{}{
			"code":    500,
			"message": "获取统计失败: " + err.Error(),
		})
		return
	}

	activeUsers, _ := dao.NewUserRepository().CountByStatus(ctx, "active")

	c.JSON(consts.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "获取成功",
		"data": map[string]interface{}{
			"total_files":     stats.TotalFiles,
			"total_users":     stats.TotalUsers,
			"total_size":      stats.TotalSize,
			"today_uploads":   stats.TodayUploads,
			"today_downloads": stats.TodayDownloads,
			"active_users":    activeUsers,
		},
	})
}

// AdminListFiles .
// @router /admin/files [GET]
func AdminListFiles(ctx context.Context, c *app.RequestContext) {
	var req admin.AdminListFilesReq
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]interface{}{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	page := normalizePage(int(req.Page), 1)
	pageSize := normalizePageSize(int(req.PageSize), 20)
	files, total, err := adminService.GetFiles(ctx, page, pageSize, strings.TrimSpace(req.Keyword))
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]interface{}{
			"code":    500,
			"message": "获取文件列表失败: " + err.Error(),
		})
		return
	}

	userRepo := dao.NewUserRepository()
	usernameCache := map[uint]string{}
	items := make([]map[string]interface{}, 0, len(files))
	for _, file := range files {
		item := buildAdminFileItem(file)
		if file.UserID != nil {
			if username, ok := usernameCache[*file.UserID]; ok {
				item["username"] = username
			} else if user, userErr := userRepo.GetByID(ctx, *file.UserID); userErr == nil {
				usernameCache[*file.UserID] = user.Username
				item["username"] = user.Username
			}
			item["user_id"] = *file.UserID
		}
		items = append(items, item)
	}

	c.JSON(consts.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "获取成功",
		"data": map[string]interface{}{
			"items":     items,
			"list":      items,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// AdminDeleteFile .
// @router /admin/files/:id [DELETE]
func AdminDeleteFile(ctx context.Context, c *app.RequestContext) {
	identifier := strings.TrimSpace(c.Param("id"))
	if identifier == "" {
		c.JSON(consts.StatusBadRequest, map[string]interface{}{
			"code":    400,
			"message": "文件标识不能为空",
		})
		return
	}

	fileRepo := dao.NewFileCodeRepository()
	var (
		file *dbmodel.FileCode
		err  error
	)

	if fileID, parseErr := strconv.ParseUint(identifier, 10, 64); parseErr == nil {
		file, err = fileRepo.GetByID(ctx, uint(fileID))
	} else {
		file, err = fileRepo.GetByCode(ctx, identifier)
	}
	if err != nil || file == nil {
		c.JSON(consts.StatusNotFound, map[string]interface{}{
			"code":    404,
			"message": "文件不存在",
		})
		return
	}

	deletePhysicalFile(ctx, file)
	if err := adminService.DeleteFile(ctx, file.ID); err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]interface{}{
			"code":    500,
			"message": "删除文件失败: " + err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "删除成功",
	})
}

// AdminListUsers .
// @router /admin/users [GET]
func AdminListUsers(ctx context.Context, c *app.RequestContext) {
	var req admin.AdminListUsersReq
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]interface{}{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	page := normalizePage(int(req.Page), 1)
	pageSize := normalizePageSize(int(req.PageSize), 20)
	users, total, err := adminService.GetUsers(ctx, page, pageSize)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]interface{}{
			"code":    500,
			"message": "获取用户列表失败: " + err.Error(),
		})
		return
	}

	items := make([]map[string]interface{}, 0, len(users))
	for _, user := range users {
		items = append(items, buildAdminUserItem(user))
	}

	c.JSON(consts.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "获取成功",
		"data": map[string]interface{}{
			"items":     items,
			"users":     items,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// AdminUpdateUserStatus .
// @router /admin/users/:id/status [PUT]
func AdminUpdateUserStatus(ctx context.Context, c *app.RequestContext) {
	var req admin.AdminUpdateUserStatusReq
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]interface{}{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	status := "inactive"
	if req.Status == 1 {
		status = "active"
	}

	if err := adminService.UpdateUserStatus(ctx, uint(req.Id), status); err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]interface{}{
			"code":    500,
			"message": "更新用户状态失败: " + err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "更新成功",
	})
}

// AdminGetConfig .
// @router /admin/config [GET]
func AdminGetConfig(ctx context.Context, c *app.RequestContext) {
	cfg := conf.GetGlobalConfig()
	if cfg == nil {
		cfg = &conf.AppConfiguration{}
	}
	shareCodeLength, shareCodeCharset := conf.GetShareCodeConfig(cfg)

	c.JSON(consts.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "获取成功",
		"data": map[string]interface{}{
			"base": map[string]interface{}{
				"name":        cfg.App.Name,
				"description": cfg.App.Description,
				"port":        cfg.Server.Port,
				"host":        cfg.Server.Host,
				"production":  cfg.App.Production,
			},
			"transfer": map[string]interface{}{
				"upload": map[string]interface{}{
					"openupload":       boolToInt(cfg.Upload.OpenUpload),
					"uploadsize":       cfg.Upload.UploadSize,
					"requirelogin":     boolToInt(cfg.Upload.RequireLogin),
					"enablechunk":      boolToInt(cfg.Upload.EnableChunk),
					"chunksize":        cfg.Upload.ChunkSize,
					"sharecodelength":  shareCodeLength,
					"sharecodecharset": shareCodeCharset,
				},
			},
			"user": map[string]interface{}{
				"allowuserregistration": boolToInt(cfg.User.AllowUserRegistration),
				"useruploadsize":        cfg.User.UserUploadSize,
				"userstoragequota":      cfg.User.UserStorageQuota,
				"sessionexpiryhours":    cfg.User.SessionExpiryHours,
			},
			"storage": map[string]interface{}{
				"type":         cfg.Storage.Type,
				"storage_path": cfg.Storage.StoragePath,
			},
		},
	})
}

// AdminUpdateConfig .
// @router /admin/config [PUT]
func AdminUpdateConfig(ctx context.Context, c *app.RequestContext) {
	var req struct {
		Config struct {
			Base struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Port        int    `json:"port"`
				Host        string `json:"host"`
				Production  bool   `json:"production"`
			} `json:"base"`
			Transfer struct {
				Upload struct {
					OpenUpload       int    `json:"openupload"`
					UploadSize       int64  `json:"uploadsize"`
					RequireLogin     int    `json:"requirelogin"`
					EnableChunk      int    `json:"enablechunk"`
					ChunkSize        int64  `json:"chunksize"`
					ShareCodeLength  int    `json:"sharecodelength"`
					ShareCodeCharset string `json:"sharecodecharset"`
				} `json:"upload"`
			} `json:"transfer"`
			User struct {
				AllowUserRegistration int   `json:"allowuserregistration"`
				UserUploadSize        int64 `json:"useruploadsize"`
				UserStorageQuota      int64 `json:"userstoragequota"`
				SessionExpiryHours    int   `json:"sessionexpiryhours"`
			} `json:"user"`
			Storage struct {
				Type        string `json:"type"`
				StoragePath string `json:"storage_path"`
			} `json:"storage"`
		} `json:"config"`
	}
	if err := c.Bind(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]interface{}{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	cfg := conf.GetGlobalConfig()
	if cfg == nil {
		cfg = &conf.AppConfiguration{}
		conf.SetGlobalConfig(cfg)
	}

	cfg.App.Name = req.Config.Base.Name
	cfg.App.Description = req.Config.Base.Description
	cfg.App.Production = req.Config.Base.Production
	if req.Config.Base.Port > 0 {
		cfg.Server.Port = req.Config.Base.Port
	}
	if strings.TrimSpace(req.Config.Base.Host) != "" {
		cfg.Server.Host = strings.TrimSpace(req.Config.Base.Host)
	}

	cfg.Upload.OpenUpload = req.Config.Transfer.Upload.OpenUpload == 1
	if req.Config.Transfer.Upload.UploadSize > 0 {
		cfg.Upload.UploadSize = req.Config.Transfer.Upload.UploadSize
	}
	cfg.Upload.RequireLogin = req.Config.Transfer.Upload.RequireLogin == 1
	cfg.Upload.EnableChunk = req.Config.Transfer.Upload.EnableChunk == 1
	if req.Config.Transfer.Upload.ChunkSize > 0 {
		cfg.Upload.ChunkSize = req.Config.Transfer.Upload.ChunkSize
	}
	cfg.Upload.ShareCodeLength = conf.NormalizeShareCodeLength(req.Config.Transfer.Upload.ShareCodeLength)
	cfg.Upload.ShareCodeCharset = conf.NormalizeShareCodeCharset(req.Config.Transfer.Upload.ShareCodeCharset)

	cfg.User.AllowUserRegistration = req.Config.User.AllowUserRegistration == 1
	if req.Config.User.UserUploadSize > 0 {
		cfg.User.UserUploadSize = req.Config.User.UserUploadSize
	}
	if req.Config.User.UserStorageQuota > 0 {
		cfg.User.UserStorageQuota = req.Config.User.UserStorageQuota
	}
	if req.Config.User.SessionExpiryHours > 0 {
		cfg.User.SessionExpiryHours = req.Config.User.SessionExpiryHours
	}

	if strings.TrimSpace(req.Config.Storage.Type) != "" {
		cfg.Storage.Type = strings.TrimSpace(req.Config.Storage.Type)
	}
	if strings.TrimSpace(req.Config.Storage.StoragePath) != "" {
		cfg.Storage.StoragePath = strings.TrimSpace(req.Config.Storage.StoragePath)
	}

	c.JSON(consts.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "保存成功",
	})
}

func normalizePage(value, fallback int) int {
	if value < 1 {
		return fallback
	}
	return value
}

func normalizePageSize(value, fallback int) int {
	if value < 1 {
		return fallback
	}
	if value > 100 {
		return 100
	}
	return value
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func formatAdminTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format("2006-01-02 15:04:05")
}

func fileDisplayName(file *dbmodel.FileCode) string {
	if file == nil {
		return ""
	}
	if file.FilePath != "" && file.Text != "" {
		return file.Text
	}
	if file.UUIDFileName != "" {
		return file.UUIDFileName
	}
	if file.Prefix != "" || file.Suffix != "" {
		return file.Prefix + file.Suffix
	}
	return ""
}

func buildAdminFileItem(file *dbmodel.FileCode) map[string]interface{} {
	isText := file.FilePath == ""
	item := map[string]interface{}{
		"id":             file.ID,
		"code":           file.Code,
		"file_name":      fileDisplayName(file),
		"uuid_file_name": fileDisplayName(file),
		"size":           file.Size,
		"file_size":      file.Size,
		"used_count":     file.UsedCount,
		"download_count": file.UsedCount,
		"created_at":     file.CreatedAt.Format("2006-01-02 15:04:05"),
		"expired_at":     formatAdminTime(file.ExpiredAt),
		"is_text_share":  isText,
		"upload_type":    map[bool]string{true: "text", false: "file"}[isText],
		"text":           "",
	}
	if isText {
		item["text"] = file.Text
		item["file_name"] = ""
		item["uuid_file_name"] = ""
	}
	return item
}

func buildAdminUserItem(user *dbmodel.UserResp) map[string]interface{} {
	quotaLimit := user.MaxStorageQuota
	if quotaLimit == 0 {
		if cfg := conf.GetGlobalConfig(); cfg != nil {
			quotaLimit = cfg.User.UserStorageQuota
		}
	}

	return map[string]interface{}{
		"id":            user.ID,
		"username":      user.Username,
		"email":         user.Email,
		"nickname":      user.Nickname,
		"role":          user.Role,
		"status":        user.Status,
		"status_code":   boolToInt(user.Status == "active"),
		"total_storage": user.TotalStorage,
		"quota_used":    user.TotalStorage,
		"quota_limit":   quotaLimit,
		"created_at":    user.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func deletePhysicalFile(ctx context.Context, file *dbmodel.FileCode) {
	if file == nil || file.FilePath == "" {
		return
	}

	storageService := storage.NewConfiguredStorage(conf.GetGlobalConfig(), "")
	_ = storageService.DeleteFile(ctx, file.GetFilePath())
}
