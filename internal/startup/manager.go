//go:build windows

// Package startup 提供开机启动功能
package startup

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// Manager 开机启动管理器
type Manager struct {
	appName string
	appPath string
}

// NewManager 创建开机启动管理器
func NewManager(appName string) *Manager {
	// 获取当前可执行文件路径
	appPath, _ := os.Executable()
	return &Manager{
		appName: appName,
		appPath: appPath,
	}
}

// Enable 启用开机启动
func (m *Manager) Enable() error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("打开注册表失败: %w", err)
	}
	defer key.Close()

	// 设置启动项，使用绝对路径
	err = key.SetStringValue(m.appName, m.appPath)
	if err != nil {
		return fmt.Errorf("设置注册表值失败: %w", err)
	}

	return nil
}

// Disable 禁用开机启动
func (m *Manager) Disable() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("打开注册表失败: %w", err)
	}
	defer key.Close()

	err = key.DeleteValue(m.appName)
	if err != nil {
		// 如果值不存在，忽略错误
		if err.Error() != "The specified value does not exist." {
			return fmt.Errorf("删除注册表值失败: %w", err)
		}
	}

	return nil
}

// IsEnabled 检查是否已启用开机启动
func (m *Manager) IsEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.READ)
	if err != nil {
		return false
	}
	defer key.Close()

	value, _, err := key.GetStringValue(m.appName)
	if err != nil {
		return false
	}

	// 检查路径是否匹配（处理可能的路径差异）
	absPath, _ := filepath.Abs(m.appPath)
	return value == absPath || value == m.appPath
}
