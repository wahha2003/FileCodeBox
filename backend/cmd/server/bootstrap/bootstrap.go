package bootstrap

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/spf13/viper"
	"github.com/zy84338719/fileCodeBox/backend/gen/http/router"
	"github.com/zy84338719/fileCodeBox/backend/internal/conf"
	"github.com/zy84338719/fileCodeBox/backend/internal/pkg/logger"
	previewPkg "github.com/zy84338719/fileCodeBox/backend/internal/preview"
	"github.com/zy84338719/fileCodeBox/backend/internal/repo/db"
	"github.com/zy84338719/fileCodeBox/backend/internal/repo/db/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Config = conf.AppConfiguration

func CORS() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		origin := string(c.GetHeader("Origin"))
		if origin == "" {
			origin = "*"
		}

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if string(c.Method()) == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next(ctx)
	}
}

func GetConfig() *Config {
	return config
}

func InitConfig(configPath string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 12345)
	v.SetDefault("server.mode", "debug")
	v.SetDefault("database.driver", "sqlite")
	v.SetDefault("database.db_name", "./data/filecodebox.db")
	v.SetDefault("user.allow_user_registration", true)
	v.SetDefault("user.require_email_verify", false)
	v.SetDefault("user.jwt_secret", "FileCodeBox2025JWT")

	if err := v.ReadInConfig(); err != nil {
		log.Printf("Warning: Failed to read config file: %v, using defaults", err)
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

func InitDatabase(config *conf.DatabaseConfig) (*gorm.DB, error) {
	if config.Driver == "sqlite" {
		dbPath := config.DBName
		if dbPath != ":memory:" {
			log.Printf("SQLite database path: %s", dbPath)
		}
	}

	if err := db.Init(config); err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	database := db.GetDB()

	log.Println("Auto migrating database tables...")
	if err := database.AutoMigrate(
		&model.User{},
		&model.FileCode{},
		&model.UploadChunk{},
		&model.TransferLog{},
		&model.AdminOperationLog{},
		&model.UserAPIKey{},
		&model.FilePreview{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Database initialized successfully")
	return database, nil
}

func CreateDefaultAdmin(database *gorm.DB) error {
	var count int64
	database.Model(&model.User{}).Where("role = ?", "admin").Count(&count)

	if count > 0 {
		log.Println("Admin user already exists")
		return nil
	}

	admin := &model.User{
		Username:     "admin",
		Email:        "admin@filecodebox.local",
		PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZRGdjGj/n3.rsQ5pPjZ5yVlWK5WAe", // password: admin123
		Nickname:     "Administrator",
		Role:         "admin",
		Status:       "active",
	}

	if err := database.Create(admin).Error; err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	log.Println("Default admin user created (username: admin, password: admin123)")
	return nil
}

var (
	database *gorm.DB
	config   *Config
)

func Bootstrap() (*server.Hertz, error) {
	var err error
	config, err = InitConfig("configs/config.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to init config: %w", err)
	}

	conf.SetGlobalConfig(config)

	loggerConfig := &logger.Config{
		Level:      config.Log.Level,
		Filename:   config.Log.Filename,
		MaxSize:    config.Log.MaxSize,
		MaxBackups: config.Log.MaxBackups,
		MaxAge:     config.Log.MaxAge,
		Compress:   config.Log.Compress,
	}
	if err := logger.Init(loggerConfig); err != nil {
		return nil, fmt.Errorf("failed to init logger: %w", err)
	}

	database, err = InitDatabase(&config.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to init database: %w", err)
	}

	if err := CreateDefaultAdmin(database); err != nil {
		logger.Error("Failed to create default admin", zap.Error(err))
	}

	if err := initPreviewService(); err != nil {
		logger.Error("Failed to init preview service", zap.Error(err))
	}

	port := config.Server.Port
	if port == 0 {
		port = 12345
	}
	h := server.New(
		server.WithHostPorts(fmt.Sprintf("%s:%d", config.Server.Host, port)),
	)

	h.Use(CORS())

	router.GeneratedRegister(h)
	customizedRegister(h)

	logger.Info("Application bootstrap completed successfully")
	return h, nil
}

func Cleanup() {
	logger.Info("Cleaning up resources...")

	if database != nil {
		if err := db.Close(); err != nil {
			logger.Error("Failed to close database", zap.Error(err))
		}
	}

	logger.Sync()
}

func customizedRegister(r *server.Hertz) {
	registerPublicRoutes(r)
	registerExtraRoutes(r)
}

func initPreviewService() error {
	previewConfig := &previewPkg.Config{
		EnablePreview:    true,
		ThumbnailWidth:   300,
		ThumbnailHeight:  200,
		MaxFileSize:      50 * 1024 * 1024,
		PreviewCachePath: "./data/previews",
		FFmpegPath:       "ffmpeg",
	}

	return previewPkg.InitService(previewConfig)
}
