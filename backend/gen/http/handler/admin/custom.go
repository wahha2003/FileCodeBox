package admin

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	dbmodel "github.com/zy84338719/fileCodeBox/backend/internal/repo/db/model"
)

// AdminTransferLogs handles the custom transfer logs route.
func AdminTransferLogs(ctx context.Context, c *app.RequestContext) {
	page := normalizePage(queryInt(c, "page"), 1)
	pageSize := normalizePageSize(queryInt(c, "page_size"), 20)
	query := dbmodel.TransferLogQuery{
		Operation: strings.TrimSpace(c.Query("operation")),
		Search:    strings.TrimSpace(c.Query("keyword")),
		Page:      page,
		PageSize:  pageSize,
	}

	logs, total, err := adminService.GetTransferLogs(ctx, query)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]interface{}{
			"code":    500,
			"message": "获取传输日志失败: " + err.Error(),
		})
		return
	}

	items := make([]map[string]interface{}, 0, len(logs))
	for _, log := range logs {
		items = append(items, map[string]interface{}{
			"id":         log.ID,
			"operation":  log.Operation,
			"file_code":  log.FileCode,
			"file_name":  log.FileName,
			"file_size":  log.FileSize,
			"username":   log.Username,
			"ip":         log.IP,
			"created_at": log.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(consts.StatusOK, map[string]interface{}{
		"code":    200,
		"message": "获取成功",
		"data": map[string]interface{}{
			"logs":      items,
			"items":     items,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

func queryInt(c *app.RequestContext, key string) int {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return 0
	}
	var parsed int
	_, _ = fmt.Sscanf(value, "%d", &parsed)
	return parsed
}
