// Package types 定义版本信息类型
package types

import "time"

// VersionInfo 版本信息
type VersionInfo struct {
	Version     string    `json:"version"`
	ReleaseDate time.Time `json:"releaseDate"`
	DownloadURL string    `json:"downloadUrl"`
	Checksum    string    `json:"checksum"`
	Size        int64     `json:"size"`
	IsLocal     bool      `json:"isLocal"`
	IsActive    bool      `json:"isActive"`
}

// LocalVersion 本地版本信息
type LocalVersion struct {
	Version     string    `json:"version"`
	Path        string    `json:"path"`
	InstalledAt time.Time `json:"installedAt"`
	IsActive    bool      `json:"isActive"`
}

// DownloadProgress 下载进度
type DownloadProgress struct {
	Version    string  `json:"version"`
	Total      int64   `json:"total"`
	Downloaded int64   `json:"downloaded"`
	Percent    float64 `json:"percent"`
	Speed      int64   `json:"speed"`
}
