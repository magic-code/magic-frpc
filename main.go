package main

import (
	"embed"
	"log"
	"runtime"

	"github.com/magic-frpc/gui/internal/app"
	"github.com/magic-frpc/gui/internal/startup"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/icons"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	appInstance := app.NewApp()

	// 初始化开机启动管理器
	appInstance.InitStartupManager(startup.NewManager("Magic FRPc"))

	// 创建 Wails v3 应用
	wailsApp := application.New(application.Options{
		Name:        "Magic FRPc",
		Description: "FRPc GUI 客户端",
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(assets),
		},
		Windows: application.WindowsOptions{
			WebviewUserDataPath: "",
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	// 注册服务
	wailsApp.RegisterService(application.NewService(appInstance))

	// 创建主窗口
	window := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      "main",
		Title:     "Magic FRPc",
		Width:     1200,
		Height:    800,
		URL:       "/",
		MinWidth:  800,
		MinHeight: 600,
	})

	// 创建系统托盘
	systemTray := wailsApp.SystemTray.New()

	// 设置托盘图标（根据平台使用不同图标）
	if runtime.GOOS == "darwin" {
		systemTray.SetTemplateIcon(icons.SystrayMacTemplate)
	} else {
		// Windows/Linux 使用深色托盘图标
		systemTray.SetIcon(icons.SystrayDark)
	}
	systemTray.SetTooltip("Magic FRPc")

	// 创建托盘菜单
	trayMenu := wailsApp.NewMenu()
	trayMenu.Add("显示窗口").OnClick(func(ctx *application.Context) {
		window.Show()
		window.Focus()
	})
	trayMenu.AddSeparator()
	trayMenu.Add("退出").OnClick(func(ctx *application.Context) {
		appInstance.OnShutdown(nil)
		wailsApp.Quit()
	})
	systemTray.SetMenu(trayMenu)

	// 点击托盘图标切换窗口显示/隐藏
	systemTray.OnClick(func() {
		if window.IsVisible() {
			window.Hide()
		} else {
			window.Show()
			window.Focus()
		}
	})

	// 监听窗口关闭事件：最小化到托盘而不是退出
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		// 检查是否启用最小化到托盘（默认启用）
		settings := appInstance.GetSettings()
		if settings.MinimizeToTray {
			// 取消关闭事件，隐藏窗口
			e.Cancel()
			window.Hide()
			log.Println("[Tray] 窗口最小化到托盘")
		} else {
			// 允许关闭，退出应用
			appInstance.OnShutdown(nil)
			wailsApp.Quit()
		}
	})

	window.Show()

	// 运行应用
	err := wailsApp.Run()
	if err != nil {
		log.Fatal(err)
	}
}
