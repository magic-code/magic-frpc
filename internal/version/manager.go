// Package version 提供版本管理功能
package version

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/magic-frpc/gui/internal/platform"
	"github.com/magic-frpc/gui/pkg/types"
)

// VersionStore 版本存储接口
type VersionStore interface {
	VersionList() ([]*types.LocalVersion, error)
	VersionSave(version, path string, isActive bool) error
	VersionSetActive(version string) error
	VersionGetActive() (*types.LocalVersion, error)
	VersionDelete(version string) error
}

// Manager 版本管理器
type Manager struct {
	githubClient *GitHubClient
	downloader   *Downloader
	extractor    *Extractor

	// 版本存储目录
	versionDir string
	// 数据库存储（可选）
	store VersionStore

	// 下载进度回调
	progressCallback func(progress *types.DownloadProgress)

	// 下载锁（防止并发下载）
	downloadMutex sync.Mutex

	// 平台信息
	platformInfo *platform.Info
}

// NewManager 创建版本管理器
func NewManager(versionDir string, store VersionStore) *Manager {
	m := &Manager{
		githubClient: NewGitHubClient(),
		versionDir:   versionDir,
		store:        store,
		platformInfo: platform.Detect(),
	}

	m.downloader = NewDownloader(versionDir)
	m.extractor = NewExtractor(versionDir)

	return m
}

// SetProgressCallback 设置下载进度回调
func (m *Manager) SetProgressCallback(callback func(progress *types.DownloadProgress)) {
	m.progressCallback = callback
	m.downloader.SetProgressCallback(callback)
}

// ListRemote 获取远程版本列表
func (m *Manager) ListRemote(ctx context.Context) ([]*types.VersionInfo, error) {
	releases, err := m.githubClient.ListReleases(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取远程版本列表失败: %w", err)
	}

	// 获取本地已安装版本
	localVersions := make(map[string]bool)
	versions, err := m.ListLocal()
	if err == nil {
		for _, v := range versions {
			localVersions[v.Version] = true
		}
	}

	// 获取当前活动版本
	activeVersion := m.GetActive()

	// 转换为版本信息列表
	result := m.githubClient.ConvertReleasesToVersionList(releases, m.platformInfo.OS, m.platformInfo.Arch)

	// 标记本地已安装和活动版本
	for _, info := range result {
		if localVersions[info.Version] {
			info.IsLocal = true
		}
		if info.Version == activeVersion {
			info.IsActive = true
		}
	}

	return result, nil
}

// Download 下载指定版本
func (m *Manager) Download(ctx context.Context, version string) error {
	m.downloadMutex.Lock()
	defer m.downloadMutex.Unlock()

	// 检查是否已下载
	if m.extractor.VersionExists(version) {
		return fmt.Errorf("版本 %s 已存在", version)
	}

	// 获取版本下载信息
	release, err := m.githubClient.GetRelease(ctx, version)
	if err != nil {
		return fmt.Errorf("获取版本信息失败: %w", err)
	}

	// 查找对应平台的资源
	var downloadURL string
	var assetSize int64

	searchPatterns := []string{
		fmt.Sprintf("%s_%s", m.platformInfo.OS, m.platformInfo.Arch),
	}

	// 处理架构映射
	if m.platformInfo.Arch == "amd64" {
		searchPatterns = append(searchPatterns, fmt.Sprintf("%s_amd64", m.platformInfo.OS))
	}

	for _, asset := range release.Assets {
		for _, pattern := range searchPatterns {
			if containsAll(asset.Name, pattern) {
				downloadURL = asset.DownloadURL
				assetSize = asset.Size
				break
			}
		}
		if downloadURL != "" {
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("未找到适用于 %s/%s 的下载资源", m.platformInfo.OS, m.platformInfo.Arch)
	}

	fmt.Printf("开始下载版本 %s，大小: %d 字节\n", version, assetSize)

	// 下载文件
	tempFile, err := m.downloader.Download(ctx, version, downloadURL)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}

	// 清理临时文件
	defer m.downloader.CleanupTempFile(tempFile)

	// 解压安装
	frpcPath, err := m.extractor.Extract(tempFile, version)
	if err != nil {
		return fmt.Errorf("解压安装失败: %w", err)
	}

	// 设置执行权限（非 Windows）
	if m.platformInfo.OS != "windows" {
		if err := os.Chmod(frpcPath, 0755); err != nil {
			return fmt.Errorf("设置执行权限失败: %w", err)
		}
	}

	// 保存版本信息到数据库
	if m.store != nil {
		if err := m.store.VersionSave(version, filepath.Dir(frpcPath), false); err != nil {
			log.Printf("保存版本信息到数据库失败: %v", err)
		}
	}

	fmt.Printf("版本 %s 安装成功: %s\n", version, frpcPath)

	return nil
}

// SetActive 设置活动版本
func (m *Manager) SetActive(version string) error {
	// 检查版本是否存在
	if !m.extractor.VersionExists(version) {
		return fmt.Errorf("版本 %s 未安装", version)
	}

	frpcPath := m.extractor.GetFrpcBinaryPath(version)

	// 优先使用数据库存储
	if m.store != nil {
		if err := m.store.VersionSave(version, filepath.Dir(frpcPath), true); err != nil {
			log.Printf("保存版本到数据库失败: %v", err)
			// 继续执行，不返回错误
		}
		return nil
	}

	// 回退到文件存储（兼容旧版本）
	activeVersionFile := filepath.Join(m.versionDir, ".active_version")
	data := []byte(version)
	if err := os.WriteFile(activeVersionFile, data, 0644); err != nil {
		return fmt.Errorf("保存活动版本失败: %w", err)
	}

	return nil
}

// GetActive 获取当前活动版本
func (m *Manager) GetActive() string {
	// 优先从数据库读取
	if m.store != nil {
		activeVersion, err := m.store.VersionGetActive()
		if err == nil && activeVersion != nil && activeVersion.Version != "" {
			return activeVersion.Version
		}
		log.Printf("从数据库获取活动版本失败: %v", err)
	}

	// 回退到文件读取（兼容旧版本）
	activeVersionFile := filepath.Join(m.versionDir, ".active_version")
	data, err := os.ReadFile(activeVersionFile)
	if err != nil {
		return ""
	}
	return string(data)
}

// GetActiveFrpcPath 获取活动版本的 frpc 路径
func (m *Manager) GetActiveFrpcPath() (string, error) {
	activeVersion := m.GetActive()
	if activeVersion == "" {
		return "", fmt.Errorf("未设置活动版本")
	}

	frpcPath := m.extractor.GetFrpcBinaryPath(activeVersion)
	if _, err := os.Stat(frpcPath); err != nil {
		return "", fmt.Errorf("活动版本 %s 的 frpc 文件不存在", activeVersion)
	}

	return frpcPath, nil
}

// ListLocal 列出本地已安装的版本
func (m *Manager) ListLocal() ([]*types.LocalVersion, error) {
	// 从文件系统获取已安装的版本
	versions, err := m.extractor.ListLocalVersions()
	if err != nil {
		return nil, err
	}

	activeVersion := m.GetActive()
	result := make([]*types.LocalVersion, 0, len(versions))

	for _, v := range versions {
		frpcPath := m.extractor.GetFrpcBinaryPath(v)
		info, err := os.Stat(frpcPath)
		installedTime := time.Now()
		if err == nil {
			installedTime = info.ModTime()
		}

		result = append(result, &types.LocalVersion{
			Version:     v,
			Path:        filepath.Dir(frpcPath),
			InstalledAt: installedTime,
			IsActive:    v == activeVersion,
		})

		// 同步到数据库（如果数据库中没有该版本）
		if m.store != nil {
			m.store.VersionSave(v, filepath.Dir(frpcPath), v == activeVersion)
		}
	}

	return result, nil
}

// Delete 删除本地版本
func (m *Manager) Delete(version string) error {
	// 不允许删除活动版本
	if m.GetActive() == version {
		return fmt.Errorf("无法删除当前活动版本")
	}

	// 删除文件系统中的版本
	if err := m.extractor.RemoveVersion(version); err != nil {
		return err
	}

	// 从数据库中删除版本记录
	if m.store != nil {
		if err := m.store.VersionDelete(version); err != nil {
			log.Printf("从数据库删除版本记录失败: %v", err)
		}
	}

	return nil
}

// GetPlatformInfo 获取平台信息
func (m *Manager) GetPlatformInfo() *platform.Info {
	return m.platformInfo
}

// ExportLocalVersionsMetadata 导出本地版本元数据为 JSON
func (m *Manager) ExportLocalVersionsMetadata() (string, error) {
	versions, err := m.ListLocal()
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(versions, "", "  ")
	if err != nil {
		return "", fmt.Errorf("序列化版本信息失败: %w", err)
	}

	return string(data), nil
}

// 辅助函数：检查字符串是否包含所有子串
func containsAll(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if substr != "" && !contains(s, substr) {
			return false
		}
	}
	return true
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && containsAt(s, substr)))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
