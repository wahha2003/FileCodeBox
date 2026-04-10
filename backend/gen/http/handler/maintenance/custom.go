package maintenance

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/zy84338719/fileCodeBox/backend/internal/conf"
	"github.com/zy84338719/fileCodeBox/backend/internal/repo/db"
)

// OptimizeDatabase handles the custom maintenance optimize route.
func OptimizeDatabase(ctx context.Context, c *app.RequestContext) {
	database := db.GetDB()
	if database == nil {
		c.JSON(consts.StatusInternalServerError, map[string]interface{}{
			"code":    500,
			"message": "数据库未初始化",
		})
		return
	}

	driver := "sqlite"
	if cfg := conf.GetGlobalConfig(); cfg != nil && cfg.Database.Driver != "" {
		driver = cfg.Database.Driver
	}

	switch driver {
	case "sqlite":
		if err := database.Exec("VACUUM").Error; err != nil {
			c.JSON(consts.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "数据库优化失败: " + err.Error(),
			})
			return
		}
		if err := database.Exec("ANALYZE").Error; err != nil {
			c.JSON(consts.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "数据库分析失败: " + err.Error(),
			})
			return
		}
	default:
		if err := database.Exec("ANALYZE").Error; err != nil {
			c.JSON(consts.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "数据库优化失败: " + err.Error(),
			})
			return
		}
	}

	c.JSON(consts.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "数据库优化完成",
	})
}
