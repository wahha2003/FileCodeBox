package bootstrap

import (
	"github.com/cloudwego/hertz/pkg/app/server"
	adminhandler "github.com/zy84338719/fileCodeBox/backend/gen/http/handler/admin"
	maintenancehandler "github.com/zy84338719/fileCodeBox/backend/gen/http/handler/maintenance"
	sharehandler "github.com/zy84338719/fileCodeBox/backend/gen/http/handler/share"
	middleware "github.com/zy84338719/fileCodeBox/backend/internal/pkg/middleware"
)

func registerExtraRoutes(r *server.Hertz) {
	r.GET("/share/user", middleware.AuthMiddleware(), sharehandler.GetUserShares)
	r.DELETE("/share/:code", middleware.AuthMiddleware(), sharehandler.DeleteShare)
	r.GET("/admin/logs/transfer", middleware.AdminMiddleware(), adminhandler.AdminTransferLogs)
	r.POST("/admin/maintenance/optimize", middleware.AdminMiddleware(), maintenancehandler.OptimizeDatabase)
}
