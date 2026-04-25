// Package platform 提供平台检测功能
package platform

import (
	"runtime"
)

// Info 平台信息
type Info struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// Detect 检测当前平台
func Detect() *Info {
	return &Info{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
}

// GetFrpcAssetName 获取 frpc 二进制资源名称
func GetFrpcAssetName(version string) string {
	info := Detect()
	var ext string
	if info.OS == "windows" {
		ext = ".zip"
	} else {
		ext = ".tar.gz"
	}

	arch := info.Arch
	if arch == "amd64" {
		arch = "amd64"
	} else if arch == "arm64" {
		arch = "arm64"
	}

	return "frp_" + version + "_" + info.OS + "_" + arch + ext
}

// GetFrpcBinaryName 获取 frpc 二进制文件名
func GetFrpcBinaryName() string {
	if runtime.GOOS == "windows" {
		return "frpc.exe"
	}
	return "frpc"
}
