// Package version 提供版本解压安装功能
package version

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Extractor 解压安装器
type Extractor struct {
	versionDir string
}

// NewExtractor 创建解压安装器
func NewExtractor(versionDir string) *Extractor {
	return &Extractor{
		versionDir: versionDir,
	}
}

// Extract 解压下载的 frp 压缩包
// archivePath: 压缩包路径
// version: 版本号
// 返回解压后的 frpc 二进制文件路径
func (e *Extractor) Extract(archivePath string, version string) (string, error) {
	// 创建版本目录
	versionPath := filepath.Join(e.versionDir, version)
	if err := os.MkdirAll(versionPath, 0755); err != nil {
		return "", fmt.Errorf("创建版本目录失败: %w", err)
	}

	// 根据文件扩展名选择解压方式
	if strings.HasSuffix(archivePath, ".zip") {
		return e.extractZip(archivePath, versionPath, version)
	} else if strings.HasSuffix(archivePath, ".tar.gz") || strings.HasSuffix(archivePath, ".tgz") {
		return e.extractTarGz(archivePath, versionPath, version)
	}

	return "", fmt.Errorf("不支持的压缩格式: %s", archivePath)
}

// extractZip 解压 ZIP 文件（Windows）
func (e *Extractor) extractZip(archivePath string, destPath string, version string) (string, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("打开 ZIP 文件失败: %w", err)
	}
	defer reader.Close()

	var frpcPath string
	frpcBinaryName := "frpc.exe"

	for _, file := range reader.File {
		// 查找 frpc 可执行文件
		fileName := filepath.Base(file.Name)
		if fileName == frpcBinaryName || fileName == "frpc" {
			// 打开压缩包内的文件
			rc, err := file.Open()
			if err != nil {
				return "", fmt.Errorf("打开压缩包内文件失败: %w", err)
			}

			// 创建目标文件
			targetPath := filepath.Join(destPath, frpcBinaryName)
			outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil {
				rc.Close()
				return "", fmt.Errorf("创建目标文件失败: %w", err)
			}

			// 复制内容
			_, err = io.Copy(outFile, rc)
			outFile.Close()
			rc.Close()

			if err != nil {
				return "", fmt.Errorf("解压文件失败: %w", err)
			}

			frpcPath = targetPath
		}
	}

	if frpcPath == "" {
		return "", fmt.Errorf("压缩包中未找到 frpc 可执行文件")
	}

	return frpcPath, nil
}

// extractTarGz 解压 tar.gz 文件（Linux/macOS）
func (e *Extractor) extractTarGz(archivePath string, destPath string, version string) (string, error) {
	// 打开 gzip 流
	file, err := os.Open(archivePath)
	if err != nil {
		return "", fmt.Errorf("打开压缩文件失败: %w", err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return "", fmt.Errorf("创建 gzip 读取器失败: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	var frpcPath string

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("读取 tar 文件失败: %w", err)
		}

		// 查找 frpc 可执行文件
		fileName := filepath.Base(header.Name)
		if fileName == "frpc" && header.Typeflag == tar.TypeReg {
			targetPath := filepath.Join(destPath, "frpc")
			outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil {
				return "", fmt.Errorf("创建目标文件失败: %w", err)
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return "", fmt.Errorf("解压文件失败: %w", err)
			}
			outFile.Close()

			frpcPath = targetPath
		}
	}

	if frpcPath == "" {
		return "", fmt.Errorf("压缩包中未找到 frpc 可执行文件")
	}

	return frpcPath, nil
}

// GetFrpcBinaryPath 获取指定版本的 frpc 二进制路径
func (e *Extractor) GetFrpcBinaryPath(version string) string {
	binaryName := "frpc"
	if runtime.GOOS == "windows" {
		binaryName = "frpc.exe"
	}
	return filepath.Join(e.versionDir, version, binaryName)
}

// VersionExists 检查版本是否已存在
func (e *Extractor) VersionExists(version string) bool {
	frpcPath := e.GetFrpcBinaryPath(version)
	_, err := os.Stat(frpcPath)
	return err == nil
}

// RemoveVersion 删除指定版本
func (e *Extractor) RemoveVersion(version string) error {
	versionPath := filepath.Join(e.versionDir, version)
	return os.RemoveAll(versionPath)
}

// ListLocalVersions 列出本地已安装的版本
func (e *Extractor) ListLocalVersions() ([]string, error) {
	entries, err := os.ReadDir(e.versionDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("读取版本目录失败: %w", err)
	}

	versions := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			// 检查目录中是否有 frpc 二进制文件
			frpcPath := e.GetFrpcBinaryPath(entry.Name())
			if _, err := os.Stat(frpcPath); err == nil {
				versions = append(versions, entry.Name())
			}
		}
	}

	return versions, nil
}
