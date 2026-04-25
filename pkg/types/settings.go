// Package types 定义应用设置类型
package types

// Settings 应用设置
type Settings struct {
	// 主题设置: light, dark, system
	Theme string `json:"theme"`
	// 语言设置: zh-CN, en-US
	Language string `json:"language"`
	// 自动重启
	AutoRestart bool `json:"autoRestart"`
	// 开机自启
	AutoStart bool `json:"autoStart"`
	// 最小化到托盘
	MinimizeToTray bool `json:"minimizeToTray"`
}

// DefaultSettings 返回默认设置
func DefaultSettings() *Settings {
	return &Settings{
		Theme:          "system",
		Language:       "zh-CN",
		AutoRestart:    false,
		AutoStart:      false,
		MinimizeToTray: true,
	}
}
