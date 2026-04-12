package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/spf13/viper"
	"github.com/zy84338719/fileCodeBox/backend/gen/http/router"
	adminsvc "github.com/zy84338719/fileCodeBox/backend/internal/app/admin"
	"github.com/zy84338719/fileCodeBox/backend/internal/conf"
	"github.com/zy84338719/fileCodeBox/backend/internal/pkg/logger"
	previewPkg "github.com/zy84338719/fileCodeBox/backend/internal/preview"
	"github.com/zy84338719/fileCodeBox/backend/internal/repo/db"
	"github.com/zy84338719/fileCodeBox/backend/internal/repo/db/model"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Config = conf.AppConfiguration

const (
	defaultAdminUsername           = "admin"
	defaultAdminPassword           = "admin123"
	defaultAdminEmail              = "admin@filecodebox.local"
	defaultAdminNickname           = "Administrator"
	legacyDefaultAdminPasswordHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZRGdjGj/n3.rsQ5pPjZ5yVlWK5WAe"
	expiredFileProbeInterval       = 10 * time.Minute
)

func CORS() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		origin := string(c.GetHeader("Origin"))
		if origin == "" {
			origin = "*"
		}

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Accept, Content-Type, Authorization, X-Requested-With, X-API-Key")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type, Content-Disposition")
		if origin != "*" {
			c.Header("Access-Control-Allow-Credentials", "true")
		}
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
	conf.SetConfigFilePath(configPath)

	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 12345)
	v.SetDefault("server.mode", "debug")
	v.SetDefault("server.base_url", "http://api.localhost:12345")
	v.SetDefault("server.public_base_url", "http://localhost:3000")
	v.SetDefault("database.driver", "sqlite")
	v.SetDefault("database.db_name", "./data/filecodebox.db")
	v.SetDefault("user.allow_user_registration", true)
	v.SetDefault("user.require_email_verify", false)
	v.SetDefault("user.jwt_secret", "FileCodeBox2025JWT")
	v.SetDefault("upload.share_code_length", conf.DefaultShareCodeLength)
	v.SetDefault("upload.share_code_charset", conf.DefaultShareCodeCharset)

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
	var existingAdmin model.User
	err := database.Where("username = ? AND role = ?", defaultAdminUsername, "admin").First(&existingAdmin).Error
	switch {
	case err == nil:
		return repairLegacyDefaultAdmin(database, &existingAdmin)
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound):
		return fmt.Errorf("failed to lookup default admin user: %w", err)
	}

	var count int64
	if err := database.Model(&model.User{}).Where("role = ?", "admin").Count(&count).Error; err != nil {
		return fmt.Errorf("failed to count admin users: %w", err)
	}

	if count > 0 {
		log.Println("Admin user already exists")
		return nil
	}

	passwordHash, err := generateDefaultAdminPasswordHash()
	if err != nil {
		return err
	}

	admin := &model.User{
		Username:     defaultAdminUsername,
		Email:        defaultAdminEmail,
		PasswordHash: passwordHash,
		Nickname:     defaultAdminNickname,
		Role:         "admin",
		Status:       "active",
	}

	if err := database.Create(admin).Error; err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	log.Printf("Default admin user created (username: %s, password: %s)", defaultAdminUsername, defaultAdminPassword)
	return nil
}

func generateDefaultAdminPasswordHash() (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(defaultAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash default admin password: %w", err)
	}

	return string(hash), nil
}

func repairLegacyDefaultAdmin(database *gorm.DB, admin *model.User) error {
	if admin.PasswordHash != legacyDefaultAdminPasswordHash {
		log.Println("Admin user already exists")
		return nil
	}

	passwordHash, err := generateDefaultAdminPasswordHash()
	if err != nil {
		return err
	}

	if err := database.Model(admin).Update("password_hash", passwordHash).Error; err != nil {
		return fmt.Errorf("failed to repair default admin user password: %w", err)
	}

	log.Printf("Default admin user password repaired (username: %s, password: %s)", defaultAdminUsername, defaultAdminPassword)
	return nil
}

var (
	database *gorm.DB
	config   *Config

	expiredCleanupCancel context.CancelFunc
	expiredCleanupDone   chan struct{}
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

	startExpiredFileCleanupWorker()

	port := config.Server.Port
	if port == 0 {
		port = 12345
	}

	maxRequestBodySize := computeMaxRequestBodySize(config)
	h := server.New(
		server.WithHostPorts(fmt.Sprintf("%s:%d", config.Server.Host, port)),
		server.WithMaxRequestBodySize(maxRequestBodySize),
		server.WithReadTimeout(time.Duration(config.Server.ReadTimeout)*time.Second),
		server.WithWriteTimeout(time.Duration(config.Server.WriteTimeout)*time.Second),
	)

	h.Use(CORS())

	router.GeneratedRegister(h)
	customizedRegister(h)

	logger.Info("Application bootstrap completed successfully")
	return h, nil
}

func Cleanup() {
	logger.Info("Cleaning up resources...")
	stopExpiredFileCleanupWorker()

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

func computeMaxRequestBodySize(cfg *conf.AppConfiguration) int {
	// Hertz defaults to 4MB, which is often smaller than this project's upload limits.
	// Add some overhead for multipart boundaries/fields.
	const multipartOverhead = 32 * 1024 * 1024 // 32MB

	var uploadLimit int64
	if cfg != nil {
		uploadLimit = max64(cfg.Upload.UploadSize, cfg.Upload.ChunkSize)
	}
	if uploadLimit <= 0 {
		uploadLimit = 10 * 1024 * 1024
	}

	requestLimit := uploadLimit + multipartOverhead
	if requestLimit > int64(math.MaxInt) {
		return math.MaxInt
	}
	return int(requestLimit)
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func startExpiredFileCleanupWorker() {
	if expiredCleanupCancel != nil {
		return
	}

	cleanupCtx, cancel := context.WithCancel(context.Background())
	expiredCleanupCancel = cancel
	expiredCleanupDone = make(chan struct{})

	service := adminsvc.NewService()

	go func() {
		defer close(expiredCleanupDone)

		probeAndCleanup := func() {
			hasExpiredFiles, err := service.HasExpiredFiles(cleanupCtx)
			if err != nil {
				if cleanupCtx.Err() != nil {
					return
				}
				logger.Error("Light probe for expired files failed", zap.Error(err))
				return
			}

			if !hasExpiredFiles {
				return
			}

			deletedCount, freedSpace, err := service.CleanExpiredFiles(cleanupCtx)
			if err != nil {
				if cleanupCtx.Err() != nil {
					return
				}
				logger.Error("Auto cleanup of expired files failed", zap.Error(err))
				return
			}

			if deletedCount > 0 {
				logger.Info(
					"Auto cleanup removed expired files",
					zap.Int64("deleted_count", deletedCount),
					zap.Int64("freed_space", freedSpace),
				)
			}
		}

		probeAndCleanup()

		ticker := time.NewTicker(expiredFileProbeInterval)
		defer ticker.Stop()

		for {
			select {
			case <-cleanupCtx.Done():
				return
			case <-ticker.C:
				probeAndCleanup()
			}
		}
	}()
}

func stopExpiredFileCleanupWorker() {
	if expiredCleanupCancel == nil {
		return
	}

	expiredCleanupCancel()
	if expiredCleanupDone != nil {
		<-expiredCleanupDone
	}

	expiredCleanupCancel = nil
	expiredCleanupDone = nil
}
