// Package version 提供 GitHub API 客户端功能
package version

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/magic-frpc/gui/pkg/types"
)

// GitHub API 常量
const (
	GitHubAPIURL      = "https://api.github.com/repos/fatedier/frp/releases"
	GitHubDownloadURL = "https://github.com/fatedier/frp/releases/download"
)

// GitHubRelease 表示 GitHub Release 信息
type GitHubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	PublishedAt time.Time     `json:"published_at"`
	Assets      []GitHubAsset `json:"assets"`
	Body        string        `json:"body"` // Release notes
}

// GitHubAsset 表示 Release 中的资源文件
type GitHubAsset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
}

// GitHubClient GitHub API 客户端
type GitHubClient struct {
	client      *http.Client
	apiURL      string
	downloadURL string
}

// NewGitHubClient 创建 GitHub API 客户端
func NewGitHubClient() *GitHubClient {
	return &GitHubClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiURL:      GitHubAPIURL,
		downloadURL: GitHubDownloadURL,
	}
}

// ListReleases 获取 frp 发布版本列表
func (c *GitHubClient) ListReleases(ctx context.Context) ([]*GitHubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL+"?per_page=50", nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置 User-Agent（GitHub API 要求）
	req.Header.Set("User-Agent", "Magic-FRPc-GUI")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 GitHub API 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API 返回错误状态码 %d: %s", resp.StatusCode, string(body))
	}

	var releases []*GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	log.Printf("从 GitHub 获取到 %d 个版本", len(releases))
	return releases, nil
}

// GetRelease 获取指定版本的发布信息
func (c *GitHubClient) GetRelease(ctx context.Context, version string) (*GitHubRelease, error) {
	// 确保 version 格式正确（以 v 开头）
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	url := fmt.Sprintf("%s/tags/%s", c.apiURL, version)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-Agent", "Magic-FRPc-GUI")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 GitHub API 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("版本 %s 不存在", version)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API 返回错误状态码 %d: %s", resp.StatusCode, string(body))
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &release, nil
}

// ConvertToVersionInfo 将 GitHubRelease 转换为 VersionInfo
func (c *GitHubClient) ConvertToVersionInfo(release *GitHubRelease, os, arch string) *types.VersionInfo {
	version := strings.TrimPrefix(release.TagName, "v")

	// 查找匹配的资源文件
	var downloadURL string
	var size int64

	// 构建匹配模式
	var patterns []string
	switch os {
	case "windows":
		// Windows 资源命名格式: frp_x.xx.x_windows_amd64.zip
		patterns = []string{
			fmt.Sprintf("windows_%s", arch),
			fmt.Sprintf("windows_%s.zip", arch),
		}
	case "darwin":
		// macOS 资源命名格式: frp_x.xx.x_darwin_arm64.tar.gz
		patterns = []string{
			fmt.Sprintf("darwin_%s", arch),
			fmt.Sprintf("darwin_%s.tar.gz", arch),
		}
	case "linux":
		// Linux 资源命名格式: frp_x.xx.x_linux_amd64.tar.gz
		patterns = []string{
			fmt.Sprintf("linux_%s", arch),
			fmt.Sprintf("linux_%s.tar.gz", arch),
		}
	}

	// 遍历所有资源查找匹配
	for _, asset := range release.Assets {
		assetName := strings.ToLower(asset.Name)
		for _, pattern := range patterns {
			if strings.Contains(assetName, strings.ToLower(pattern)) {
				downloadURL = asset.DownloadURL
				size = asset.Size
				log.Printf("版本 %s 找到匹配资源: %s -> %s", version, asset.Name, pattern)
				break
			}
		}
		if downloadURL != "" {
			break
		}
	}

	return &types.VersionInfo{
		Version:     version,
		ReleaseDate: release.PublishedAt,
		DownloadURL: downloadURL,
		Size:        size,
		IsLocal:     false,
		IsActive:    false,
	}
}

// ConvertReleasesToVersionList 批量转换发布列表
func (c *GitHubClient) ConvertReleasesToVersionList(releases []*GitHubRelease, os, arch string) []*types.VersionInfo {
	result := make([]*types.VersionInfo, 0, len(releases))
	for _, r := range releases {
		info := c.ConvertToVersionInfo(r, os, arch)
		// 只添加有下载链接的版本
		if info.DownloadURL != "" {
			result = append(result, info)
		} else {
			log.Printf("版本 %s 没有匹配的 %s/%s 资源，跳过", info.Version, os, arch)
		}
	}
	return result
}
