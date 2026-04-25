// Package app 提供应用生命周期管理
package app

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ServiceStartup 接口由 Wails v3 定义
// App 需要实现此接口以在应用启动时初始化
var _ interface {
	ServiceStartup(ctx context.Context, options application.ServiceOptions) error
} = (*App)(nil)
