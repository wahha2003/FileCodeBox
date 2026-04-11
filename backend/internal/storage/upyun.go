package storage

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/upyun/go-sdk/v3/upyun"
)

// UpyunStorage 又拍云存储实现
type UpyunStorage struct {
	config  *StorageConfig
	client  *upyun.UpYun
	initErr error
}

// NewUpyunStorage 创建又拍云存储实例
func NewUpyunStorage(config *StorageConfig) *UpyunStorage {
	s := &UpyunStorage{config: config}
	s.init()
	return s
}

func (s *UpyunStorage) init() {
	bucket := strings.TrimSpace(s.config.UpyunBucket)
	operator := strings.TrimSpace(s.config.UpyunOperator)
	password := strings.TrimSpace(s.config.UpyunPassword)

	if bucket == "" || operator == "" || password == "" {
		s.initErr = fmt.Errorf("又拍云 Bucket/Operator/Password 未配置")
		return
	}

	s.client = upyun.NewUpYun(&upyun.UpYunConfig{
		Bucket:   bucket,
		Operator: operator,
		Password: password,
	})
}

func (s *UpyunStorage) ensureReady() error {
	if s.initErr != nil {
		return s.initErr
	}
	if s.client == nil {
		return fmt.Errorf("又拍云客户端未初始化")
	}
	return nil
}

// ==================== StorageInterface 基础操作 ====================

func (s *UpyunStorage) SaveFile(ctx context.Context, file *multipart.FileHeader, savePath string) (*FileOperationResult, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}

	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("打开上传文件失败: %w", err)
	}
	defer src.Close()

	key := "/" + s.normalizeKey(savePath)
	err = s.client.Put(&upyun.PutObjectConfig{
		Path:   key,
		Reader: src,
	})
	if err != nil {
		return nil, fmt.Errorf("上传文件到又拍云失败: %w", err)
	}

	return &FileOperationResult{
		Success:   true,
		Message:   "文件上传成功",
		FilePath:  s.normalizeKey(savePath),
		FileSize:  file.Size,
		Timestamp: time.Now(),
	}, nil
}

func (s *UpyunStorage) DeleteFile(ctx context.Context, filePath string) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	return s.client.Delete(&upyun.DeleteObjectConfig{
		Path: "/" + s.normalizeKey(filePath),
	})
}

func (s *UpyunStorage) GetFile(ctx context.Context, filePath string) ([]byte, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	_, err := s.client.Get(&upyun.GetObjectConfig{
		Path:   "/" + s.normalizeKey(filePath),
		Writer: &buf,
	})
	if err != nil {
		return nil, fmt.Errorf("从又拍云下载文件失败: %w", err)
	}
	return buf.Bytes(), nil
}

func (s *UpyunStorage) FileExists(ctx context.Context, filePath string) bool {
	if err := s.ensureReady(); err != nil {
		return false
	}
	_, err := s.client.GetInfo("/" + s.normalizeKey(filePath))
	return err == nil
}

func (s *UpyunStorage) SaveChunk(ctx context.Context, uploadID string, chunkIndex int, data []byte) error {
	// 服务端 fallback：将分片作为独立对象保存
	if err := s.ensureReady(); err != nil {
		return err
	}

	key := fmt.Sprintf("/chunks/%s/%d.part", uploadID, chunkIndex)
	err := s.client.Put(&upyun.PutObjectConfig{
		Path:   key,
		Reader: bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("保存分片到又拍云失败: %w", err)
	}
	return nil
}

func (s *UpyunStorage) MergeChunks(ctx context.Context, uploadID string, totalChunks int, savePath string) error {
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

	key := "/" + s.normalizeKey(savePath)
	uploadErr := s.client.Put(&upyun.PutObjectConfig{
		Path:   key,
		Reader: pipeReader,
	})

	streamErr := <-errCh
	if uploadErr != nil {
		return fmt.Errorf("合并上传到又拍云失败: %w", uploadErr)
	}
	if streamErr != nil {
		return streamErr
	}

	go s.CleanChunks(context.Background(), uploadID)
	return nil
}

func (s *UpyunStorage) CleanChunks(ctx context.Context, uploadID string) error {
	if err := s.ensureReady(); err != nil {
		return err
	}

	prefix := fmt.Sprintf("/chunks/%s/", uploadID)

	// 列出并删除所有分片
	objsChan := make(chan *upyun.FileInfo, 100)
	go func() {
		_ = s.client.List(&upyun.GetObjectsConfig{
			Path:         prefix,
			ObjectsChan:  objsChan,
		})
	}()

	for obj := range objsChan {
		_ = s.client.Delete(&upyun.DeleteObjectConfig{
			Path: prefix + obj.Name,
		})
	}

	// 删除目录本身
	_ = s.client.Delete(&upyun.DeleteObjectConfig{
		Path: prefix,
	})

	return nil
}

func (s *UpyunStorage) GetFileSize(ctx context.Context, filePath string) (int64, error) {
	if err := s.ensureReady(); err != nil {
		return 0, err
	}
	info, err := s.client.GetInfo("/" + s.normalizeKey(filePath))
	if err != nil {
		return 0, fmt.Errorf("获取又拍云文件信息失败: %w", err)
	}
	return info.Size, nil
}

func (s *UpyunStorage) GetFileURL(ctx context.Context, filePath string) (string, error) {
	domain := strings.TrimSpace(s.config.UpyunDomain)
	if domain == "" {
		return "", fmt.Errorf("又拍云 CDN 域名未配置")
	}

	key := s.normalizeKey(filePath)
	rawURL := fmt.Sprintf("https://%s/%s", domain, key)

	// 如果配置了 Token 防盗链密钥，生成签名 URL
	secret := strings.TrimSpace(s.config.UpyunSecret)
	if secret != "" {
		etime := time.Now().Add(time.Hour).Unix()
		signStr := fmt.Sprintf("%s&%d&%s", secret, etime, "/"+key)
		h := md5.New()
		h.Write([]byte(signStr))
		token := fmt.Sprintf("%x", h.Sum(nil))
		// 取中间8位
		if len(token) > 8 {
			token = token[12:20]
		}
		rawURL = fmt.Sprintf("%s?_upt=%s%d", rawURL, token, etime)
	}

	return rawURL, nil
}

func (s *UpyunStorage) GetFileReader(ctx context.Context, filePath string) (io.ReadCloser, int64, error) {
	if err := s.ensureReady(); err != nil {
		return nil, 0, err
	}

	// 如果有 CDN 域名，通过 HTTP 下载
	downloadURL, err := s.GetFileURL(ctx, filePath)
	if err == nil && downloadURL != "" {
		resp, httpErr := http.Get(downloadURL)
		if httpErr == nil && resp.StatusCode == http.StatusOK {
			return resp.Body, resp.ContentLength, nil
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	// fallback: 通过 SDK 下载
	pr, pw := io.Pipe()
	go func() {
		_, getErr := s.client.Get(&upyun.GetObjectConfig{
			Path:   "/" + s.normalizeKey(filePath),
			Writer: pw,
		})
		if getErr != nil {
			_ = pw.CloseWithError(getErr)
		} else {
			_ = pw.Close()
		}
	}()

	// 获取文件大小
	size, _ := s.GetFileSize(ctx, filePath)
	return pr, size, nil
}

func (s *UpyunStorage) TestConnection(ctx context.Context) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	_, err := s.client.Usage()
	if err != nil {
		return fmt.Errorf("又拍云连接测试失败: %w", err)
	}
	return nil
}

// ==================== DirectUploader 接口实现（sign_proxy 签名代理模式） ====================

// InitiateMultipartUpload 初始化又拍云分片上传
// 服务端调用又拍 REST API 发起分片上传，拿到 X-Upyun-Multi-UUID
func (s *UpyunStorage) InitiateMultipartUpload(ctx context.Context, objectKey string) (string, error) {
	if err := s.ensureReady(); err != nil {
		return "", err
	}

	key := "/" + s.normalizeKey(objectKey)

	// 构造 initiate 请求
	url := fmt.Sprintf("https://v0.api.upyun.com/%s%s", s.config.UpyunBucket, key)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("X-Upyun-Multi-Stage", "initiate")
	req.Header.Set("X-Upyun-Multi-Type", "application/octet-stream")

	// 签名
	s.signRequest(req, key)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("初始化又拍云分片上传失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("初始化又拍云分片上传失败，状态码 %d: %s", resp.StatusCode, string(body))
	}

	multiUUID := resp.Header.Get("X-Upyun-Multi-Uuid")
	if multiUUID == "" {
		return "", fmt.Errorf("又拍云未返回 Multi-UUID")
	}

	return multiUUID, nil
}

// GenerateUploadPartURLs 又拍云不支持批量预签名，返回空
func (s *UpyunStorage) GenerateUploadPartURLs(ctx context.Context, objectKey, platformUploadID string, totalParts int) ([]PresignedPart, error) {
	return nil, nil
}

// GenerateUploadPartAuth 为单个分片生成签名认证信息（核心方法）
// 服务端计算 Authorization，客户端携带签名直传到又拍云
func (s *UpyunStorage) GenerateUploadPartAuth(ctx context.Context, objectKey, platformUploadID string, partIndex, partCount int) (*PartAuthInfo, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}

	key := "/" + s.normalizeKey(objectKey)
	uploadURL := fmt.Sprintf("https://v0.api.upyun.com/%s%s", s.config.UpyunBucket, key)

	// 构建需要签名的请求头
	headers := map[string]string{
		"X-Upyun-Multi-Stage": "upload",
		"X-Upyun-Multi-Uuid":  platformUploadID,
		"X-Upyun-Part-Id":     fmt.Sprintf("%d", partIndex),
	}

	// 计算 Authorization
	// 又拍云签名格式: UPYUN operator:signature
	// signature = Base64(HMAC-SHA1(MD5(password), method&uri&date))
	date := time.Now().UTC().Format(http.TimeFormat)
	headers["Date"] = date

	method := "PUT"
	uri := fmt.Sprintf("/%s%s", s.config.UpyunBucket, key)

	passwordMD5 := fmt.Sprintf("%x", md5.Sum([]byte(s.config.UpyunPassword)))
	stringToSign := fmt.Sprintf("%s&%s&%s", method, uri, date)
	mac := hmac.New(sha1.New, []byte(passwordMD5))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	headers["Authorization"] = fmt.Sprintf("UPYUN %s:%s", s.config.UpyunOperator, signature)

	return &PartAuthInfo{
		URL:     uploadURL,
		Headers: headers,
		Method:  "PUT",
	}, nil
}

// CompleteMultipartUpload 完成又拍云分片上传
func (s *UpyunStorage) CompleteMultipartUpload(ctx context.Context, objectKey, platformUploadID string, parts []CompletedPart) error {
	if err := s.ensureReady(); err != nil {
		return err
	}

	key := "/" + s.normalizeKey(objectKey)
	url := fmt.Sprintf("https://v0.api.upyun.com/%s%s", s.config.UpyunBucket, key)

	req, err := http.NewRequestWithContext(ctx, "PUT", url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("X-Upyun-Multi-Stage", "complete")
	req.Header.Set("X-Upyun-Multi-Uuid", platformUploadID)

	// 签名
	s.signRequest(req, key)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("完成又拍云分片上传失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("完成又拍云分片上传失败，状态码 %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// AbortMultipartUpload 又拍云不支持显式取消，分片会在24h后自动清理
func (s *UpyunStorage) AbortMultipartUpload(ctx context.Context, objectKey, platformUploadID string) error {
	// 又拍云不支持显式取消分片上传，未完成的分片会在 24 小时后自动清理
	return nil
}

// GeneratePresignedPutURL 生成 Form API 签名（小文件直传）
func (s *UpyunStorage) GeneratePresignedPutURL(ctx context.Context, objectKey, contentType string) (string, error) {
	if err := s.ensureReady(); err != nil {
		return "", err
	}

	// 又拍云 Form API 上传 URL
	return fmt.Sprintf("https://v0.api.upyun.com/%s", s.config.UpyunBucket), nil
}

// GeneratePresignedGetURL 生成预签名下载 URL
func (s *UpyunStorage) GeneratePresignedGetURL(ctx context.Context, objectKey string) (string, error) {
	return s.GetFileURL(ctx, objectKey)
}

// SupportsDirectUpload 又拍云支持直传
func (s *UpyunStorage) SupportsDirectUpload() bool {
	return true
}

// DirectUploadMode 又拍云使用签名代理模式
func (s *UpyunStorage) DirectUploadMode() string {
	return DirectUploadModeSignProxy
}

// ==================== 内部方法 ====================

func (s *UpyunStorage) normalizeKey(key string) string {
	key = strings.TrimSpace(key)
	key = strings.ReplaceAll(key, "\\", "/")
	return strings.TrimPrefix(key, "/")
}

// signRequest 为请求添加又拍云签名
func (s *UpyunStorage) signRequest(req *http.Request, path string) {
	date := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("Date", date)

	method := req.Method
	uri := fmt.Sprintf("/%s%s", s.config.UpyunBucket, path)

	passwordMD5 := fmt.Sprintf("%x", md5.Sum([]byte(s.config.UpyunPassword)))
	stringToSign := fmt.Sprintf("%s&%s&%s", method, uri, date)
	mac := hmac.New(sha1.New, []byte(passwordMD5))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	req.Header.Set("Authorization", fmt.Sprintf("UPYUN %s:%s", s.config.UpyunOperator, signature))
}

// 确保 UpyunStorage 实现了接口
var _ StorageInterface = (*UpyunStorage)(nil)
var _ DirectUploader = (*UpyunStorage)(nil)
var _ ConnectionTester = (*UpyunStorage)(nil)
