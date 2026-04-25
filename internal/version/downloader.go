// Package version 提供下载管理功能
package version

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/magic-frpc/gui/pkg/types"
)

// Downloader 下载管理器
type Downloader struct {
	client     *http.Client
	versionDir string
	// 下载进度回调
	progressCallback func(progress *types.DownloadProgress)
}

// NewDownloader 创建下载管理器
func NewDownloader(versionDir string) *Downloader {
	return &Downloader{
		client: &http.Client{
			Timeout: 30 * time.Minute, // 下载可能需要较长时间
		},
		versionDir: versionDir,
	}
}

// SetProgressCallback 设置进度回调
func (d *Downloader) SetProgressCallback(callback func(progress *types.DownloadProgress)) {
	d.progressCallback = callback
}

// Download 下载指定版本的 frpc
func (d *Downloader) Download(ctx context.Context, version string, downloadURL string) (string, error) {
	// 确保版本目录存在
	if err := os.MkdirAll(d.versionDir, 0755); err != nil {
		return "", fmt.Errorf("创建版本目录失败: %w", err)
	}

	// 从 URL 中提取文件扩展名
	ext := ".tmp"
	if idx := strings.LastIndex(downloadURL, "."); idx > 0 {
		urlExt := downloadURL[idx:]
		// 检查是否是压缩文件扩展名
		if strings.Contains(urlExt, "zip") || strings.Contains(urlExt, "tar.gz") || strings.Contains(urlExt, "tgz") {
			ext = urlExt
			// 移除可能的查询参数
			if qIdx := strings.Index(ext, "?"); qIdx > 0 {
				ext = ext[:qIdx]
			}
		}
	}

	// 创建临时文件（保留扩展名以便解压时识别格式）
	tempFile := filepath.Join(d.versionDir, fmt.Sprintf("frp_%s%s", version, ext))
	out, err := os.Create(tempFile)
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer out.Close()

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建下载请求失败: %w", err)
	}

	req.Header.Set("User-Agent", "Magic-FRPc-GUI")
	req.Header.Set("Accept", "*/*")

	// 发起请求
	resp, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	// 下载进度跟踪
	total := resp.ContentLength
	var downloaded int64
	var lastTime time.Time
	var lastDownloaded int64

	buf := make([]byte, 32*1024) // 32KB buffer
	progress := &types.DownloadProgress{
		Version: version,
		Total:   total,
	}

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				return "", fmt.Errorf("写入文件失败: %w", writeErr)
			}

			downloaded += int64(n)
			progress.Downloaded = downloaded

			// 计算百分比
			if total > 0 {
				progress.Percent = float64(downloaded) / float64(total) * 100
			}

			// 计算下载速度（每秒更新一次）
			now := time.Now()
			if lastTime.IsZero() {
				lastTime = now
				lastDownloaded = downloaded
			} else if now.Sub(lastTime) >= time.Second {
				elapsed := now.Sub(lastTime).Seconds()
				progress.Speed = int64(float64(downloaded-lastDownloaded) / elapsed)
				lastTime = now
				lastDownloaded = downloaded
			}

			// 触发进度回调
			if d.progressCallback != nil {
				d.progressCallback(progress)
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("读取响应失败: %w", err)
		}
	}

	// 最终进度
	progress.Percent = 100
	progress.Speed = 0
	if d.progressCallback != nil {
		d.progressCallback(progress)
	}

	return tempFile, nil
}

// VerifySHA256 验证文件 SHA256
func (d *Downloader) VerifySHA256(filePath string, expectedHash string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("计算哈希失败: %w", err)
	}

	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if expectedHash != "" && actualHash != expectedHash {
		return fmt.Errorf("SHA256 校验失败: 期望 %s, 实际 %s", expectedHash, actualHash)
	}

	return nil
}

// CalculateSHA256 计算文件的 SHA256 哈希
func (d *Downloader) CalculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("计算哈希失败: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// CleanupTempFile 清理临时文件
func (d *Downloader) CleanupTempFile(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(filePath)
}
