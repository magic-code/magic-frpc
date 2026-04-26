//go:build !windows

// Package startup 提供开机启动功能
package startup

// Manager 开机启动管理器
type Manager struct {
	appName string
}

// NewManager 创建开机启动管理器
func NewManager(appName string) *Manager {
	return &Manager{
		appName: appName,
	}
}

// Enable 在非 Windows 平台上不执行任何操作
func (m *Manager) Enable() error {
	return nil
}

// Disable 在非 Windows 平台上不执行任何操作
func (m *Manager) Disable() error {
	return nil
}

// IsEnabled 在非 Windows 平台上始终返回 false
func (m *Manager) IsEnabled() bool {
	return false
}
