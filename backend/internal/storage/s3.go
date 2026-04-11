package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	aws "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	s3service "github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const (
	defaultS3Region   = "us-east-1"
	defaultSignedTTL  = time.Hour
	s3ChunkPrefixRoot = "chunks"
)

// S3Storage 基于 S3 兼容协议的对象存储实现。
type S3Storage struct {
	config    *StorageConfig
	client    *s3service.Client
	presigner *s3service.PresignClient
	clientErr error
}

// NewS3Storage 创建 S3 存储实例。
func NewS3Storage(config *StorageConfig) *S3Storage {
	storage := &S3Storage{config: config}
	client, presigner, err := storage.newClient()
	storage.client = client
	storage.presigner = presigner
	storage.clientErr = err
	return storage
}

func (s *S3Storage) SaveFile(ctx context.Context, file *multipart.FileHeader, savePath string) (*FileOperationResult, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}

	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("打开上传文件失败: %w", err)
	}
	defer src.Close()

	contentType := file.Header.Get("Content-Type")
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}

	hasher := sha256.New()
	key := s.normalizeObjectKey(savePath)
	body := io.TeeReader(src, hasher)

	_, err = s.client.PutObject(ctx, &s3service.PutObjectInput{
		Bucket:        aws.String(s.config.Bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentLength: aws.Int64(file.Size),
		ContentType:   aws.String(contentType),
	})
	if err != nil {
		return nil, fmt.Errorf("上传文件到 S3 失败: %w", err)
	}

	return &FileOperationResult{
		Success:   true,
		Message:   "文件上传成功",
		FilePath:  key,
		FileSize:  file.Size,
		FileHash:  hex.EncodeToString(hasher.Sum(nil)),
		Timestamp: time.Now(),
	}, nil
}

func (s *S3Storage) DeleteFile(ctx context.Context, filePath string) error {
	if err := s.ensureReady(); err != nil {
		return err
	}

	_, err := s.client.DeleteObject(ctx, &s3service.DeleteObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(s.normalizeObjectKey(filePath)),
	})
	if err != nil {
		return fmt.Errorf("删除 S3 文件失败: %w", err)
	}
	return nil
}

func (s *S3Storage) GetFile(ctx context.Context, filePath string) ([]byte, error) {
	reader, _, err := s.GetFileReader(ctx, filePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("读取 S3 文件失败: %w", err)
	}
	return data, nil
}

func (s *S3Storage) FileExists(ctx context.Context, filePath string) bool {
	if err := s.ensureReady(); err != nil {
		return false
	}

	_, err := s.client.HeadObject(ctx, &s3service.HeadObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(s.normalizeObjectKey(filePath)),
	})
	return err == nil
}

func (s *S3Storage) SaveChunk(ctx context.Context, uploadID string, chunkIndex int, data []byte) error {
	if err := s.ensureReady(); err != nil {
		return err
	}

	chunkKey := s.chunkObjectKey(uploadID, chunkIndex)
	_, err := s.client.PutObject(ctx, &s3service.PutObjectInput{
		Bucket:        aws.String(s.config.Bucket),
		Key:           aws.String(chunkKey),
		Body:          bytes.NewReader(data),
		ContentLength: aws.Int64(int64(len(data))),
		ContentType:   aws.String("application/octet-stream"),
		Metadata: map[string]string{
			"chunk-index": fmt.Sprintf("%d", chunkIndex),
		},
	})
	if err != nil {
		return fmt.Errorf("上传 S3 分片失败: %w", err)
	}
	return nil
}

func (s *S3Storage) MergeChunks(ctx context.Context, uploadID string, totalChunks int, savePath string) error {
	if err := s.ensureReady(); err != nil {
		return err
	}

	pipeReader, pipeWriter := io.Pipe()
	streamErrCh := make(chan error, 1)

	go func() {
		for i := 0; i < totalChunks; i++ {
			result, err := s.client.GetObject(ctx, &s3service.GetObjectInput{
				Bucket: aws.String(s.config.Bucket),
				Key:    aws.String(s.chunkObjectKey(uploadID, i)),
			})
			if err != nil {
				streamErrCh <- fmt.Errorf("读取分片 %d 失败: %w", i, err)
				_ = pipeWriter.CloseWithError(err)
				return
			}

			_, copyErr := io.Copy(pipeWriter, result.Body)
			closeErr := result.Body.Close()
			if copyErr != nil {
				streamErrCh <- fmt.Errorf("写入合并流失败: %w", copyErr)
				_ = pipeWriter.CloseWithError(copyErr)
				return
			}
			if closeErr != nil {
				streamErrCh <- fmt.Errorf("关闭分片流失败: %w", closeErr)
				_ = pipeWriter.CloseWithError(closeErr)
				return
			}
		}

		streamErrCh <- pipeWriter.Close()
	}()

	uploader := manager.NewUploader(s.client, func(u *manager.Uploader) {
		u.PartSize = 5 * 1024 * 1024
		u.LeavePartsOnError = false
	})

	_, err := uploader.Upload(ctx, &s3service.PutObjectInput{
		Bucket:      aws.String(s.config.Bucket),
		Key:         aws.String(s.normalizeObjectKey(savePath)),
		Body:        pipeReader,
		ContentType: aws.String("application/octet-stream"),
	})
	streamErr := <-streamErrCh
	if err != nil {
		return fmt.Errorf("合并分片上传到 S3 失败: %w", err)
	}
	if streamErr != nil {
		return streamErr
	}

	go s.CleanChunks(context.Background(), uploadID)
	return nil
}

func (s *S3Storage) CleanChunks(ctx context.Context, uploadID string) error {
	if err := s.ensureReady(); err != nil {
		return err
	}

	prefix := s.normalizeObjectKey(strings.TrimSuffix(s.chunkDirPrefix(uploadID), "/") + "/")
	paginator := s3service.NewListObjectsV2Paginator(s.client, &s3service.ListObjectsV2Input{
		Bucket: aws.String(s.config.Bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("列出 S3 分片失败: %w", err)
		}
		if len(page.Contents) == 0 {
			continue
		}

		objects := make([]s3types.ObjectIdentifier, 0, len(page.Contents))
		for _, item := range page.Contents {
			if item.Key == nil {
				continue
			}
			objects = append(objects, s3types.ObjectIdentifier{Key: item.Key})
		}
		if len(objects) == 0 {
			continue
		}

		_, err = s.client.DeleteObjects(ctx, &s3service.DeleteObjectsInput{
			Bucket: aws.String(s.config.Bucket),
			Delete: &s3types.Delete{Objects: objects, Quiet: aws.Bool(true)},
		})
		if err != nil {
			return fmt.Errorf("删除 S3 分片失败: %w", err)
		}
	}

	return nil
}

func (s *S3Storage) GetFileSize(ctx context.Context, filePath string) (int64, error) {
	if err := s.ensureReady(); err != nil {
		return 0, err
	}

	result, err := s.client.HeadObject(ctx, &s3service.HeadObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(s.normalizeObjectKey(filePath)),
	})
	if err != nil {
		return 0, fmt.Errorf("获取 S3 文件信息失败: %w", err)
	}
	return aws.ToInt64(result.ContentLength), nil
}

func (s *S3Storage) GetFileURL(ctx context.Context, filePath string) (string, error) {
	if err := s.ensureReady(); err != nil {
		return "", err
	}

	result, err := s.presigner.PresignGetObject(ctx, &s3service.GetObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(s.normalizeObjectKey(filePath)),
	}, func(opts *s3service.PresignOptions) {
		opts.Expires = defaultSignedTTL
	})
	if err != nil {
		return "", fmt.Errorf("生成 S3 预签名下载链接失败: %w", err)
	}
	return result.URL, nil
}

func (s *S3Storage) GetFileReader(ctx context.Context, filePath string) (io.ReadCloser, int64, error) {
	if err := s.ensureReady(); err != nil {
		return nil, 0, err
	}

	result, err := s.client.GetObject(ctx, &s3service.GetObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(s.normalizeObjectKey(filePath)),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("从 S3 获取文件失败: %w", err)
	}

	return result.Body, aws.ToInt64(result.ContentLength), nil
}

func (s *S3Storage) TestConnection(ctx context.Context) error {
	if err := s.ensureReady(); err != nil {
		return err
	}

	_, err := s.client.HeadBucket(ctx, &s3service.HeadBucketInput{
		Bucket: aws.String(s.config.Bucket),
	})
	if err != nil {
		return fmt.Errorf("S3 连接测试失败: %w", err)
	}
	return nil
}

func (s *S3Storage) ensureReady() error {
	if s.config == nil {
		return fmt.Errorf("S3 配置为空")
	}
	if strings.TrimSpace(s.config.AccessKey) == "" {
		return fmt.Errorf("S3 Access Key 未配置")
	}
	if strings.TrimSpace(s.config.SecretKey) == "" {
		return fmt.Errorf("S3 Secret Key 未配置")
	}
	if strings.TrimSpace(s.config.Bucket) == "" {
		return fmt.Errorf("S3 Bucket 未配置")
	}
	if s.clientErr != nil {
		return s.clientErr
	}
	if s.client == nil || s.presigner == nil {
		return fmt.Errorf("S3 客户端初始化失败")
	}
	return nil
}

func (s *S3Storage) newClient() (*s3service.Client, *s3service.PresignClient, error) {
	region := strings.TrimSpace(s.config.Region)
	if region == "" {
		region = defaultS3Region
	}

	loadOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			strings.TrimSpace(s.config.AccessKey),
			strings.TrimSpace(s.config.SecretKey),
			"",
		)),
	}

	if httpClient, err := s.newHTTPClient(); err != nil {
		return nil, nil, err
	} else if httpClient != nil {
		loadOptions = append(loadOptions, awsconfig.WithHTTPClient(httpClient))
	}

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(), loadOptions...)
	if err != nil {
		return nil, nil, fmt.Errorf("加载 S3 SDK 配置失败: %w", err)
	}

	client := s3service.NewFromConfig(cfg, func(o *s3service.Options) {
		if endpoint := s.endpointURL(); endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		}
	})

	return client, s3service.NewPresignClient(client), nil
}

func (s *S3Storage) newHTTPClient() (*http.Client, error) {
	proxyValue := strings.TrimSpace(s.config.Proxy)
	if proxyValue == "" {
		return nil, nil
	}

	proxyURL, err := url.Parse(proxyValue)
	if err != nil {
		return nil, fmt.Errorf("S3 代理地址无效: %w", err)
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyURL(proxyURL)
	return &http.Client{Transport: transport}, nil
}

func (s *S3Storage) endpointURL() string {
	if endpoint := strings.TrimSpace(s.config.Endpoint); endpoint != "" {
		return endpoint
	}

	hostname := strings.TrimSpace(s.config.Hostname)
	if hostname == "" {
		return ""
	}
	if strings.HasPrefix(hostname, "http://") || strings.HasPrefix(hostname, "https://") {
		return hostname
	}
	return "https://" + hostname
}

func (s *S3Storage) chunkDirPrefix(uploadID string) string {
	return s.normalizeObjectKey(strings.Join([]string{s3ChunkPrefixRoot, uploadID}, "/"))
}

func (s *S3Storage) chunkObjectKey(uploadID string, chunkIndex int) string {
	return s.normalizeObjectKey(fmt.Sprintf("%s/%d.part", s.chunkDirPrefix(uploadID), chunkIndex))
}

func (s *S3Storage) normalizeObjectKey(key string) string {
	key = strings.TrimSpace(key)
	key = strings.ReplaceAll(key, "\\", "/")
	return strings.TrimPrefix(key, "/")
}

// ==================== DirectUploader 接口实现（presigned 模式） ====================

// InitiateMultipartUpload 初始化 S3 原生分片上传
func (s *S3Storage) InitiateMultipartUpload(ctx context.Context, objectKey string) (string, error) {
	if err := s.ensureReady(); err != nil {
		return "", err
	}

	result, err := s.client.CreateMultipartUpload(ctx, &s3service.CreateMultipartUploadInput{
		Bucket:      aws.String(s.config.Bucket),
		Key:         aws.String(s.normalizeObjectKey(objectKey)),
		ContentType: aws.String("application/octet-stream"),
	})
	if err != nil {
		return "", fmt.Errorf("初始化 S3 分片上传失败: %w", err)
	}
	return aws.ToString(result.UploadId), nil
}

// GenerateUploadPartURLs 批量生成分片上传预签名 URL
func (s *S3Storage) GenerateUploadPartURLs(ctx context.Context, objectKey, platformUploadID string, totalParts int) ([]PresignedPart, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}

	expiry := s.getSignedExpiry()
	parts := make([]PresignedPart, 0, totalParts)
	key := s.normalizeObjectKey(objectKey)

	for i := 1; i <= totalParts; i++ {
		result, err := s.presigner.PresignUploadPart(ctx, &s3service.UploadPartInput{
			Bucket:     aws.String(s.config.Bucket),
			Key:        aws.String(key),
			UploadId:   aws.String(platformUploadID),
			PartNumber: aws.Int32(int32(i)),
		}, func(opts *s3service.PresignOptions) {
			opts.Expires = expiry
		})
		if err != nil {
			return nil, fmt.Errorf("生成分片 %d 预签名 URL 失败: %w", i, err)
		}
		parts = append(parts, PresignedPart{
			PartNumber: i,
			URL:        result.URL,
		})
	}
	return parts, nil
}

// GenerateUploadPartAuth S3 不需要签名代理模式，返回错误
func (s *S3Storage) GenerateUploadPartAuth(ctx context.Context, objectKey, platformUploadID string, partIndex, partCount int) (*PartAuthInfo, error) {
	return nil, fmt.Errorf("S3 存储不支持签名代理模式，请使用 presigned 模式")
}

// CompleteMultipartUpload 完成 S3 原生分片上传
func (s *S3Storage) CompleteMultipartUpload(ctx context.Context, objectKey, platformUploadID string, parts []CompletedPart) error {
	if err := s.ensureReady(); err != nil {
		return err
	}

	s3Parts := make([]s3types.CompletedPart, 0, len(parts))
	for _, p := range parts {
		s3Parts = append(s3Parts, s3types.CompletedPart{
			PartNumber: aws.Int32(int32(p.PartNumber)),
			ETag:       aws.String(p.ETag),
		})
	}

	_, err := s.client.CompleteMultipartUpload(ctx, &s3service.CompleteMultipartUploadInput{
		Bucket:   aws.String(s.config.Bucket),
		Key:      aws.String(s.normalizeObjectKey(objectKey)),
		UploadId: aws.String(platformUploadID),
		MultipartUpload: &s3types.CompletedMultipartUpload{
			Parts: s3Parts,
		},
	})
	if err != nil {
		return fmt.Errorf("完成 S3 分片上传失败: %w", err)
	}
	return nil
}

// AbortMultipartUpload 取消 S3 原生分片上传
func (s *S3Storage) AbortMultipartUpload(ctx context.Context, objectKey, platformUploadID string) error {
	if err := s.ensureReady(); err != nil {
		return err
	}

	_, err := s.client.AbortMultipartUpload(ctx, &s3service.AbortMultipartUploadInput{
		Bucket:   aws.String(s.config.Bucket),
		Key:      aws.String(s.normalizeObjectKey(objectKey)),
		UploadId: aws.String(platformUploadID),
	})
	if err != nil {
		return fmt.Errorf("取消 S3 分片上传失败: %w", err)
	}
	return nil
}

// GeneratePresignedPutURL 生成小文件直传预签名 PUT URL
func (s *S3Storage) GeneratePresignedPutURL(ctx context.Context, objectKey, contentType string) (string, error) {
	if err := s.ensureReady(); err != nil {
		return "", err
	}

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	result, err := s.presigner.PresignPutObject(ctx, &s3service.PutObjectInput{
		Bucket:      aws.String(s.config.Bucket),
		Key:         aws.String(s.normalizeObjectKey(objectKey)),
		ContentType: aws.String(contentType),
	}, func(opts *s3service.PresignOptions) {
		opts.Expires = s.getSignedExpiry()
	})
	if err != nil {
		return "", fmt.Errorf("生成预签名 PUT URL 失败: %w", err)
	}
	return result.URL, nil
}

// GeneratePresignedGetURL 生成预签名下载 URL
func (s *S3Storage) GeneratePresignedGetURL(ctx context.Context, objectKey string) (string, error) {
	return s.GetFileURL(ctx, objectKey)
}

// SupportsDirectUpload S3 支持直传
func (s *S3Storage) SupportsDirectUpload() bool {
	return true
}

// DirectUploadMode S3 使用 presigned 模式
func (s *S3Storage) DirectUploadMode() string {
	return DirectUploadModePresigned
}

// getSignedExpiry 获取预签名 URL 有效期
func (s *S3Storage) getSignedExpiry() time.Duration {
	if s.config.SignedURLExpiry > 0 {
		return time.Duration(s.config.SignedURLExpiry) * time.Second
	}
	return defaultSignedTTL
}

