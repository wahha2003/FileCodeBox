package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/qiniu/go-sdk/v7/auth"
	qiniustorage "github.com/qiniu/go-sdk/v7/storage"
	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/http_client"
	"github.com/qiniu/go-sdk/v7/storagev2/uploader"
	"github.com/qiniu/go-sdk/v7/storagev2/uptoken"
)

// QiniuStorage 七牛云存储实现
type QiniuStorage struct {
	config    *StorageConfig
	mac       *auth.Credentials
	cred      *credentials.Credentials
	bucketMgr *qiniustorage.BucketManager
	uploadMgr *uploader.UploadManager
	initErr   error
}

// NewQiniuStorage 创建七牛云存储实例
func NewQiniuStorage(config *StorageConfig) *QiniuStorage {
	s := &QiniuStorage{config: config}
	s.init()
	return s
}

func (s *QiniuStorage) init() {
	ak := strings.TrimSpace(s.config.QiniuAccessKey)
	sk := strings.TrimSpace(s.config.QiniuSecretKey)
	if ak == "" || sk == "" {
		s.initErr = fmt.Errorf("七牛云 AccessKey/SecretKey 未配置")
		return
	}

	s.mac = auth.New(ak, sk)
	s.cred = credentials.NewCredentials(ak, sk)

	// 使用 v1 API 的 BucketManager 做文件管理
	cfg := qiniustorage.Config{
		UseHTTPS: s.config.QiniuUseHTTPS,
	}
	s.bucketMgr = qiniustorage.NewBucketManager(s.mac, &cfg)

	// 使用 v2 API 的 UploadManager 做上传
	s.uploadMgr = uploader.NewUploadManager(&uploader.UploadManagerOptions{
		Options: http_client.Options{
			Credentials: s.cred,
		},
	})
}

func (s *QiniuStorage) ensureReady() error {
	if s.initErr != nil {
		return s.initErr
	}
	if strings.TrimSpace(s.config.QiniuBucket) == "" {
		return fmt.Errorf("七牛云 Bucket 未配置")
	}
	return nil
}

// ==================== StorageInterface 基础操作 ====================

func (s *QiniuStorage) SaveFile(ctx context.Context, file *multipart.FileHeader, savePath string) (*FileOperationResult, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}

	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("打开上传文件失败: %w", err)
	}
	defer src.Close()

	key := s.normalizeKey(savePath)
	objectName := key

	putPolicy, err := uptoken.NewPutPolicy(s.config.QiniuBucket, time.Now().Add(time.Hour))
	if err != nil {
		return nil, fmt.Errorf("创建上传策略失败: %w", err)
	}
	signer := uptoken.NewSigner(putPolicy, s.cred)

	err = s.uploadMgr.UploadReader(ctx, src, &uploader.ObjectOptions{
		BucketName: s.config.QiniuBucket,
		ObjectName: &objectName,
		UpToken:    signer,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("上传文件到七牛云失败: %w", err)
	}

	return &FileOperationResult{
		Success:   true,
		Message:   "文件上传成功",
		FilePath:  key,
		FileSize:  file.Size,
		Timestamp: time.Now(),
	}, nil
}

func (s *QiniuStorage) DeleteFile(ctx context.Context, filePath string) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	return s.bucketMgr.Delete(s.config.QiniuBucket, s.normalizeKey(filePath))
}

func (s *QiniuStorage) GetFile(ctx context.Context, filePath string) ([]byte, error) {
	reader, _, err := s.GetFileReader(ctx, filePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func (s *QiniuStorage) FileExists(ctx context.Context, filePath string) bool {
	if err := s.ensureReady(); err != nil {
		return false
	}
	_, err := s.bucketMgr.Stat(s.config.QiniuBucket, s.normalizeKey(filePath))
	return err == nil
}

func (s *QiniuStorage) SaveChunk(ctx context.Context, uploadID string, chunkIndex int, data []byte) error {
	// 服务端 fallback：将分片作为独立对象保存
	if err := s.ensureReady(); err != nil {
		return err
	}

	key := fmt.Sprintf("chunks/%s/%d.part", uploadID, chunkIndex)
	objectName := key

	putPolicy, err := uptoken.NewPutPolicy(s.config.QiniuBucket, time.Now().Add(time.Hour))
	if err != nil {
		return fmt.Errorf("创建上传策略失败: %w", err)
	}
	signer := uptoken.NewSigner(putPolicy, s.cred)

	err = s.uploadMgr.UploadReader(ctx, bytes.NewReader(data), &uploader.ObjectOptions{
		BucketName: s.config.QiniuBucket,
		ObjectName: &objectName,
		UpToken:    signer,
	}, nil)
	if err != nil {
		return fmt.Errorf("保存分片到七牛云失败: %w", err)
	}
	return nil
}

func (s *QiniuStorage) MergeChunks(ctx context.Context, uploadID string, totalChunks int, savePath string) error {
	// 服务端 fallback：下载所有分片再合并上传
	if err := s.ensureReady(); err != nil {
		return err
	}

	pipeReader, pipeWriter := io.Pipe()
	errCh := make(chan error, 1)

	go func() {
		for i := 0; i < totalChunks; i++ {
			chunkKey := fmt.Sprintf("chunks/%s/%d.part", uploadID, i)
			data, err := s.GetFile(ctx, chunkKey)
			if err != nil {
				errCh <- fmt.Errorf("读取分片 %d 失败: %w", i, err)
				_ = pipeWriter.CloseWithError(err)
				return
			}
			if _, err := pipeWriter.Write(data); err != nil {
				errCh <- fmt.Errorf("写入合并流失败: %w", err)
				_ = pipeWriter.CloseWithError(err)
				return
			}
		}
		errCh <- pipeWriter.Close()
	}()

	key := s.normalizeKey(savePath)
	objectName := key

	putPolicy, err := uptoken.NewPutPolicy(s.config.QiniuBucket, time.Now().Add(2*time.Hour))
	if err != nil {
		return fmt.Errorf("创建上传策略失败: %w", err)
	}
	signer := uptoken.NewSigner(putPolicy, s.cred)

	uploadErr := s.uploadMgr.UploadReader(ctx, pipeReader, &uploader.ObjectOptions{
		BucketName: s.config.QiniuBucket,
		ObjectName: &objectName,
		UpToken:    signer,
	}, nil)

	streamErr := <-errCh
	if uploadErr != nil {
		return fmt.Errorf("合并上传到七牛云失败: %w", uploadErr)
	}
	if streamErr != nil {
		return streamErr
	}

	go s.CleanChunks(context.Background(), uploadID)
	return nil
}

func (s *QiniuStorage) CleanChunks(ctx context.Context, uploadID string) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	// 列出并删除所有分片
	prefix := fmt.Sprintf("chunks/%s/", uploadID)
	entries, _, _, _, err := s.bucketMgr.ListFiles(s.config.QiniuBucket, prefix, "", "", 1000)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		_ = s.bucketMgr.Delete(s.config.QiniuBucket, entry.Key)
	}
	return nil
}

func (s *QiniuStorage) GetFileSize(ctx context.Context, filePath string) (int64, error) {
	if err := s.ensureReady(); err != nil {
		return 0, err
	}
	info, err := s.bucketMgr.Stat(s.config.QiniuBucket, s.normalizeKey(filePath))
	if err != nil {
		return 0, fmt.Errorf("获取七牛云文件信息失败: %w", err)
	}
	return info.Fsize, nil
}

func (s *QiniuStorage) GetFileURL(ctx context.Context, filePath string) (string, error) {
	domain := strings.TrimSpace(s.config.QiniuDomain)
	if domain == "" {
		return "", fmt.Errorf("七牛云 CDN 域名未配置")
	}

	key := s.normalizeKey(filePath)
	scheme := "https"
	if !s.config.QiniuUseHTTPS {
		scheme = "http"
	}

	rawURL := fmt.Sprintf("%s://%s/%s", scheme, domain, key)

	// 私有空间需要签名
	if s.config.QiniuPrivate {
		deadline := time.Now().Add(time.Hour).Unix()
		return qiniustorage.MakePrivateURL(s.mac, domain, key, deadline), nil
	}

	return rawURL, nil
}

func (s *QiniuStorage) GetFileReader(ctx context.Context, filePath string) (io.ReadCloser, int64, error) {
	downloadURL, err := s.GetFileURL(ctx, filePath)
	if err != nil {
		return nil, 0, err
	}

	resp, err := http.Get(downloadURL)
	if err != nil {
		return nil, 0, fmt.Errorf("从七牛云下载文件失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, 0, fmt.Errorf("从七牛云下载文件失败，状态码: %d", resp.StatusCode)
	}

	return resp.Body, resp.ContentLength, nil
}

func (s *QiniuStorage) TestConnection(ctx context.Context) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	_, err := s.bucketMgr.GetBucketInfo(s.config.QiniuBucket)
	if err != nil {
		return fmt.Errorf("七牛云连接测试失败: %w", err)
	}
	return nil
}

// ==================== DirectUploader 接口实现（presigned 模式） ====================

// InitiateMultipartUpload 七牛云使用 UpToken 机制，此处生成一个标识
func (s *QiniuStorage) InitiateMultipartUpload(ctx context.Context, objectKey string) (string, error) {
	if err := s.ensureReady(); err != nil {
		return "", err
	}
	// 七牛云客户端直传不需要服务端初始化分片，返回一个时间戳作为标识
	return fmt.Sprintf("qiniu_%d", time.Now().UnixNano()), nil
}

// GenerateUploadPartURLs 生成带 UpToken 的分片上传 URL
// 七牛云客户端直传使用 UpToken 机制，每个分片可以独立上传
func (s *QiniuStorage) GenerateUploadPartURLs(ctx context.Context, objectKey, platformUploadID string, totalParts int) ([]PresignedPart, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}

	// 七牛云客户端直传：生成 UpToken，前端使用 JS SDK 上传
	putPolicy := qiniustorage.PutPolicy{
		Scope:   fmt.Sprintf("%s:%s", s.config.QiniuBucket, s.normalizeKey(objectKey)),
		Expires: 7200, // 2小时有效
	}
	upToken := putPolicy.UploadToken(s.mac)

	// 获取上传域名
	region, err := s.getUploadRegion()
	if err != nil {
		return nil, err
	}

	parts := make([]PresignedPart, 0, totalParts)
	for i := 1; i <= totalParts; i++ {
		// 每个分片共用同一个 UpToken，前端按照七牛分片上传协议上传
		parts = append(parts, PresignedPart{
			PartNumber: i,
			URL:        fmt.Sprintf("%s?uptoken=%s&partNumber=%d", region, upToken, i),
		})
	}
	return parts, nil
}

// GenerateUploadPartAuth 七牛使用 presigned 模式，不需要签名代理
func (s *QiniuStorage) GenerateUploadPartAuth(ctx context.Context, objectKey, platformUploadID string, partIndex, partCount int) (*PartAuthInfo, error) {
	return nil, fmt.Errorf("七牛云使用 presigned 模式，不支持签名代理")
}

// CompleteMultipartUpload 七牛云分片上传完成
func (s *QiniuStorage) CompleteMultipartUpload(ctx context.Context, objectKey, platformUploadID string, parts []CompletedPart) error {
	// 七牛云客户端直传完成后，由客户端 JS SDK 调用 complete
	// 服务端只需要确认文件存在即可
	return nil
}

// AbortMultipartUpload 取消七牛云分片上传
func (s *QiniuStorage) AbortMultipartUpload(ctx context.Context, objectKey, platformUploadID string) error {
	// 七牛云自动清理未完成的分片
	return nil
}

// GeneratePresignedPutURL 生成小文件直传上传凭证
func (s *QiniuStorage) GeneratePresignedPutURL(ctx context.Context, objectKey, contentType string) (string, error) {
	if err := s.ensureReady(); err != nil {
		return "", err
	}

	putPolicy := qiniustorage.PutPolicy{
		Scope:   fmt.Sprintf("%s:%s", s.config.QiniuBucket, s.normalizeKey(objectKey)),
		Expires: 3600,
	}
	upToken := putPolicy.UploadToken(s.mac)

	region, err := s.getUploadRegion()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s?uptoken=%s", region, upToken), nil
}

// GeneratePresignedGetURL 生成预签名下载 URL
func (s *QiniuStorage) GeneratePresignedGetURL(ctx context.Context, objectKey string) (string, error) {
	return s.GetFileURL(ctx, objectKey)
}

// SupportsDirectUpload 七牛云支持直传
func (s *QiniuStorage) SupportsDirectUpload() bool {
	return true
}

// DirectUploadMode 七牛云使用 presigned 模式
func (s *QiniuStorage) DirectUploadMode() string {
	return DirectUploadModePresigned
}

// ==================== 内部方法 ====================

func (s *QiniuStorage) normalizeKey(key string) string {
	key = strings.TrimSpace(key)
	key = strings.ReplaceAll(key, "\\", "/")
	return strings.TrimPrefix(key, "/")
}

// getUploadRegion 获取上传域名
func (s *QiniuStorage) getUploadRegion() (string, error) {
	regionStr := strings.TrimSpace(s.config.QiniuRegion)
	scheme := "https"
	if !s.config.QiniuUseHTTPS {
		scheme = "http"
	}

	// 七牛云各区域上传域名
	regionMap := map[string]string{
		"z0":         "up-z0.qiniup.com",
		"z1":         "up-z1.qiniup.com",
		"z2":         "up-z2.qiniup.com",
		"na0":        "up-na0.qiniup.com",
		"as0":        "up-as0.qiniup.com",
		"cn-east-2":  "up-cn-east-2.qiniup.com",
	}

	host, ok := regionMap[regionStr]
	if !ok {
		// 默认使用华东区域
		host = "up-z0.qiniup.com"
	}

	return fmt.Sprintf("%s://%s", scheme, host), nil
}

// 确保 QiniuStorage 实现了接口
var _ StorageInterface = (*QiniuStorage)(nil)
var _ DirectUploader = (*QiniuStorage)(nil)
var _ ConnectionTester = (*QiniuStorage)(nil)
