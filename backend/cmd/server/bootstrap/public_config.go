package bootstrap

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/zy84338719/fileCodeBox/backend/internal/conf"
)

type publicConfigData struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	UploadSize       int64    `json:"uploadSize"`
	UserUploadSize   int64    `json:"userUploadSize"`
	EnableChunk      int      `json:"enableChunk"`
	OpenUpload       int      `json:"openUpload"`
	RequireLogin     int      `json:"requireLogin"`
	ExpireStyle      []string `json:"expireStyle"`
	ShareCodeLength  int      `json:"shareCodeLength"`
	ShareCodeCharset string   `json:"shareCodeCharset"`
}

func registerPublicRoutes(r *server.Hertz) {
	r.GET("/config", getPublicConfig)
}

func getPublicConfig(ctx context.Context, c *app.RequestContext) {
	cfg := conf.GetGlobalConfig()
	if cfg == nil {
		cfg = &conf.AppConfiguration{}
	}
	shareCodeLength, shareCodeCharset := conf.GetShareCodeConfig(cfg)

	c.JSON(consts.StatusOK, map[string]any{
		"code":    200,
		"message": "success",
		"data": publicConfigData{
			Name:             firstNonEmpty(cfg.App.Name, "FileCodeBox"),
			Description:      firstNonEmpty(cfg.App.Description, "安全、便捷的文件分享系统"),
			UploadSize:       cfg.Upload.UploadSize,
			UserUploadSize:   cfg.User.UserUploadSize,
			EnableChunk:      boolToInt(cfg.Upload.EnableChunk),
			OpenUpload:       boolToInt(cfg.Upload.OpenUpload),
			RequireLogin:     boolToInt(cfg.Upload.RequireLogin),
			ExpireStyle:      []string{"minute", "hour", "day", "week", "month", "year", "forever"},
			ShareCodeLength:  shareCodeLength,
			ShareCodeCharset: shareCodeCharset,
		},
	})
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
