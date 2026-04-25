// Package app 提供 Wails 应用绑定服务
package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/magic-frpc/gui/internal/config"
	"github.com/magic-frpc/gui/internal/frpc"
	"github.com/magic-frpc/gui/internal/platform"
	"github.com/magic-frpc/gui/internal/startup"
	"github.com/magic-frpc/gui/internal/store"
	"github.com/magic-frpc/gui/internal/version"
	"github.com/magic-frpc/gui/pkg/types"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// App 是主应用服务，提供所有 API 绑定
type App struct {
	// 应用生命周期上下文
	ctx context.Context
	// 应用数据目录
	dataDir string
	// 版本管理器
	versionManager *version.Manager
	// 配置管理器
	configManager *config.Manager
	// SQLite 存储
	sqliteStore *store.SQLiteStore
	// 进程管理器（按配置 ID 管理）
	processManagers map[string]*frpc.ProcessManager
	processMu       sync.RWMutex
	// 应用日志
	appLogs   []*types.LogEntry
	appLogsMu sync.RWMutex
	// 开机启动管理器
	startupManager *startup.Manager
}

// NewApp 创建新的应用实例
func NewApp() *App {
	return &App{
		processManagers: make(map[string]*frpc.ProcessManager),
		appLogs:         make([]*types.LogEntry, 0),
	}
}

// Name 返回服务名称
func (a *App) Name() string {
	return "App"
}

// ServiceStartup 应用启动时调用（Wails v3 接口）
func (a *App) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	a.ctx = ctx

	// 初始化应用数据目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("获取用户目录失败: %v", err)
		return fmt.Errorf("获取用户目录失败: %v", err)
	}

	a.dataDir = filepath.Join(homeDir, ".magic-frpc")
	if err := os.MkdirAll(a.dataDir, 0755); err != nil {
		log.Printf("创建数据目录失败: %v", err)
		return fmt.Errorf("创建数据目录失败: %v", err)
	}

	// 初始化 SQLite 存储
	dbPath := filepath.Join(a.dataDir, "magic-frpc.db")
	a.sqliteStore, err = store.NewSQLiteStore(dbPath)
	if err != nil {
		log.Printf("初始化数据库失败: %v", err)
		return fmt.Errorf("初始化数据库失败: %v", err)
	}

	// 初始化配置管理器
	a.configManager = config.NewManager(a.sqliteStore)

	// 初始化版本管理器
	versionDir := filepath.Join(a.dataDir, "versions")
	a.versionManager = version.NewManager(versionDir, a.sqliteStore)

	// 添加启动日志
	a.addAppLog("info", "应用启动成功")
	a.addAppLog("info", fmt.Sprintf("数据目录: %s", a.dataDir))

	log.Printf("应用启动成功，数据目录: %s", a.dataDir)

	// 自动启动标记为 AutoStartOnLaunch 的配置
	go a.autoStartMarkedConfigs()

	return nil
}

// OnShutdown 应用关闭时调用
func (a *App) OnShutdown(ctx context.Context) error {
	log.Println("应用正在关闭...")

	// 停止所有进程
	a.processMu.Lock()
	for id, pm := range a.processManagers {
		if err := pm.Stop(); err != nil {
			log.Printf("停止进程 %s 失败: %v", id, err)
		}
	}
	a.processMu.Unlock()

	// 关闭数据库连接
	if a.sqliteStore != nil {
		if err := a.sqliteStore.Close(); err != nil {
			log.Printf("关闭数据库失败: %v", err)
		}
	}

	return nil
}

// GetDataDir 获取应用数据目录
func (a *App) GetDataDir() string {
	return a.dataDir
}

// Greet 示例 API - 返回问候语
func (a *App) Greet(name string) string {
	return "Hello " + name + "! Welcome to Magic FRPc."
}

// ========== 配置管理 API ==========

// ConfigList 获取配置列表
func (a *App) ConfigList() ([]*types.ConfigMeta, error) {
	if a.configManager == nil {
		return []*types.ConfigMeta{}, fmt.Errorf("配置管理器未初始化")
	}
	return a.configManager.List()
}

// ConfigGet 获取配置详情
func (a *App) ConfigGet(id string) (*types.FrpcConfig, error) {
	if a.configManager == nil {
		return nil, fmt.Errorf("配置管理器未初始化")
	}
	return a.configManager.Get(id)
}

// ConfigSave 保存配置
func (a *App) ConfigSave(configJSON string) error {
	if a.configManager == nil {
		return fmt.Errorf("配置管理器未初始化")
	}
	var cfg types.FrpcConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return err
	}

	log.Printf("保存配置: ID=%s, Name=%s, Proxies=%d", cfg.ID, cfg.Name, len(cfg.Proxies))
	for i, p := range cfg.Proxies {
		log.Printf("  代理[%d]: Name=%s, Type=%s, LocalPort=%d, RemotePort=%d", i, p.Name, p.Type, p.LocalPort, p.RemotePort)
	}

	_, err := a.configManager.Save(&cfg)
	return err
}

// ConfigDelete 删除配置
func (a *App) ConfigDelete(id string) error {
	if a.configManager == nil {
		return fmt.Errorf("配置管理器未初始化")
	}
	return a.configManager.Delete(id)
}

// ConfigValidate 验证配置
func (a *App) ConfigValidate(configJSON string) (*config.ValidationResult, error) {
	if a.configManager == nil {
		return nil, fmt.Errorf("配置管理器未初始化")
	}
	var cfg types.FrpcConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, err
	}

	return a.configManager.Validate(&cfg), nil
}

// ConfigNew 创建新配置
func (a *App) ConfigNew(name string) (*types.FrpcConfig, error) {
	if a.configManager == nil {
		return nil, fmt.Errorf("配置管理器未初始化")
	}
	config := a.configManager.NewConfig(name)
	// 直接保存到数据库，跳过验证（新配置可以后续编辑）
	if err := a.sqliteStore.ConfigSave(config); err != nil {
		return nil, fmt.Errorf("保存新配置失败: %w", err)
	}
	return config, nil
}

// ConfigImport 导入配置文件
func (a *App) ConfigImport(name string, data string, format string) (*types.FrpcConfig, error) {
	if a.configManager == nil {
		return nil, fmt.Errorf("配置管理器未初始化")
	}
	return a.configManager.Import(name, []byte(data), format)
}

// ConfigExport 导出配置文件
func (a *App) ConfigExport(id string, format string) (map[string]string, error) {
	if a.configManager == nil {
		return nil, fmt.Errorf("配置管理器未初始化")
	}
	data, filename, err := a.configManager.Export(id, format)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"content":  string(data),
		"filename": filename,
	}, nil
}

// ConfigSerialize 序列化配置到指定格式
func (a *App) ConfigSerialize(configJSON string, format string) (string, error) {
	if a.configManager == nil {
		return "", fmt.Errorf("配置管理器未初始化")
	}
	var cfg types.FrpcConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return "", err
	}

	data, err := a.configManager.SerializeConfig(&cfg, format)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// ConfigParse 从源码解析配置
func (a *App) ConfigParse(source string, format string) (*types.FrpcConfig, error) {
	if a.configManager == nil {
		return nil, fmt.Errorf("配置管理器未初始化")
	}
	return a.configManager.ParseConfig([]byte(source), format)
}

// ========== 设置 API ==========

// GetSettings 获取应用设置
func (a *App) GetSettings() *types.Settings {
	settings := types.DefaultSettings()

	// 从数据库加载设置
	if a.sqliteStore != nil {
		if theme, err := a.sqliteStore.GetSetting("theme"); err == nil && theme != "" {
			settings.Theme = theme
		}
		if lang, err := a.sqliteStore.GetSetting("language"); err == nil && lang != "" {
			settings.Language = lang
		}
		if autoStart, err := a.sqliteStore.GetSetting("autoStart"); err == nil && autoStart == "true" {
			settings.AutoStart = true
		}
		if autoRestart, err := a.sqliteStore.GetSetting("autoRestart"); err == nil && autoRestart == "true" {
			settings.AutoRestart = true
		}
		if minimizeToTray, err := a.sqliteStore.GetSetting("minimizeToTray"); err == nil {
			// 默认为 true，只有明确设为 false 才是 false
			settings.MinimizeToTray = minimizeToTray != "false"
		}
	}

	return settings
}

// SaveSettings 保存应用设置
func (a *App) SaveSettings(settingsJSON string) error {
	var settings types.Settings
	if err := json.Unmarshal([]byte(settingsJSON), &settings); err != nil {
		return err
	}

	if a.sqliteStore != nil {
		if err := a.sqliteStore.SetSetting("theme", settings.Theme); err != nil {
			return err
		}
		if err := a.sqliteStore.SetSetting("language", settings.Language); err != nil {
			return err
		}
		// 保存开机启动设置，并同步更新注册表
		autoStartStr := "false"
		if settings.AutoStart {
			autoStartStr = "true"
		}
		if err := a.sqliteStore.SetSetting("autoStart", autoStartStr); err != nil {
			return err
		}
		// 同步更新注册表
		if a.startupManager != nil {
			if settings.AutoStart {
				a.startupManager.Enable()
			} else {
				a.startupManager.Disable()
			}
		}

		autoRestartStr := "false"
		if settings.AutoRestart {
			autoRestartStr = "true"
		}
		if err := a.sqliteStore.SetSetting("autoRestart", autoRestartStr); err != nil {
			return err
		}

		minimizeToTrayStr := "true"
		if !settings.MinimizeToTray {
			minimizeToTrayStr = "false"
		}
		if err := a.sqliteStore.SetSetting("minimizeToTray", minimizeToTrayStr); err != nil {
			return err
		}
	}

	return nil
}

// GetDB 获取数据库连接
func (a *App) GetDB() *sql.DB {
	if a.sqliteStore == nil {
		return nil
	}
	return a.sqliteStore.DB()
}

// ==================== 版本管理 API ====================

// VersionListRemote 获取远程版本列表
func (a *App) VersionListRemote() ([]*types.VersionInfo, error) {
	if a.versionManager == nil {
		return []*types.VersionInfo{}, fmt.Errorf("版本管理器未初始化")
	}
	return a.versionManager.ListRemote(a.ctx)
}

// VersionDownload 下载指定版本
func (a *App) VersionDownload(version string) error {
	if a.versionManager == nil {
		return fmt.Errorf("版本管理器未初始化")
	}

	// 设置进度回调，通过事件发送到前端
	a.versionManager.SetProgressCallback(func(progress *types.DownloadProgress) {
		// 发送事件到前端
		application.Get().Event.Emit("download-progress", map[string]interface{}{
			"version":    progress.Version,
			"total":      progress.Total,
			"downloaded": progress.Downloaded,
			"percent":    progress.Percent,
			"speed":      progress.Speed,
		})
	})

	return a.versionManager.Download(a.ctx, version)
}

// VersionSetActive 设置活动版本
func (a *App) VersionSetActive(version string) error {
	if a.versionManager == nil {
		return fmt.Errorf("版本管理器未初始化")
	}
	return a.versionManager.SetActive(version)
}

// VersionGetActive 获取当前活动版本
func (a *App) VersionGetActive() string {
	if a.versionManager == nil {
		return ""
	}
	return a.versionManager.GetActive()
}

// VersionListLocal 获取本地已安装版本列表
func (a *App) VersionListLocal() ([]*types.LocalVersion, error) {
	if a.versionManager == nil {
		return []*types.LocalVersion{}, nil
	}
	return a.versionManager.ListLocal()
}

// VersionDelete 删除本地版本
func (a *App) VersionDelete(version string) error {
	if a.versionManager == nil {
		return fmt.Errorf("版本管理器未初始化")
	}
	return a.versionManager.Delete(version)
}

// VersionGetActiveFrpcPath 获取活动版本的 frpc 路径
func (a *App) VersionGetActiveFrpcPath() (string, error) {
	if a.versionManager == nil {
		return "", fmt.Errorf("版本管理器未初始化")
	}
	return a.versionManager.GetActiveFrpcPath()
}

// PlatformGetInfo 获取平台信息
func (a *App) PlatformGetInfo() *platform.Info {
	if a.versionManager == nil {
		return &platform.Info{OS: "windows", Arch: "amd64"}
	}
	return a.versionManager.GetPlatformInfo()
}

// ==================== 进程管理 API ====================

// FrpcStart 启动 frpc 进程
func (a *App) FrpcStart(configID string) error {
	if a.configManager == nil {
		return fmt.Errorf("配置管理器未初始化")
	}
	if a.versionManager == nil {
		return fmt.Errorf("版本管理器未初始化")
	}

	// 获取配置
	cfg, err := a.configManager.Get(configID)
	if err != nil {
		return err
	}
	if cfg == nil {
		return fmt.Errorf("配置不存在: %s", configID)
	}

	// 获取活动版本的 frpc 路径
	frpcPath, err := a.versionManager.GetActiveFrpcPath()
	if err != nil {
		return err
	}

	// Dashboard 配置（本地监控服务，不影响代理功能）
	if cfg.Common == nil {
		cfg.Common = &types.CommonSection{}
	}
	// Dashboard 地址始终使用本地地址（仅用于本地监控）
	if cfg.Common.AdminAddr == "" {
		cfg.Common.AdminAddr = cfg.Common.ServerAddr
	}
	// Dashboard 端口：使用配置值，默认 7500
	if cfg.Common.AdminPort == 0 {
		cfg.Common.AdminPort = 7500
	}

	// 从配置中提取代理列表
	proxyList := make([]types.ProxyStatus, 0, len(cfg.Proxies))
	for _, p := range cfg.Proxies {
		proxyList = append(proxyList, types.ProxyStatus{
			Name:   p.Name,
			Type:   p.Type,
			Status: "starting", // 初始状态为启动中
		})
	}

	// 序列化配置为 TOML 格式
	configContent, err := a.configManager.SerializeConfig(cfg, "toml")
	if err != nil {
		return err
	}

	log.Printf("[FrpcStart] 配置 %s 包含 %d 个代理", configID, len(proxyList))

	// 获取或创建进程管理器
	a.processMu.Lock()
	defer a.processMu.Unlock()

	pm, exists := a.processManagers[configID]
	if !exists {
		pm = frpc.NewProcessManager()
		pm.SetFrpcPath(frpcPath)
		// 设置事件回调
		pm.SetCallback(func(eventType string, data interface{}) {
			log.Printf("[Process Event] type=%s, data=%v", eventType, data)
		})
		a.processManagers[configID] = pm
	}

	// 设置代理列表
	pm.SetProxies(proxyList)
	// 设置 Dashboard 配置（用于流量统计）
	pm.SetDashboard(cfg.Common.AdminAddr, cfg.Common.AdminPort, cfg.Common.AdminUser, cfg.Common.AdminPassword)

	return pm.Start(a.ctx, configID, string(configContent))
}

// allocateDashboardPort 分配 Dashboard 端口
func (a *App) allocateDashboardPort() int {
	a.processMu.Lock()
	defer a.processMu.Unlock()

	// 收集已使用的端口
	usedPorts := make(map[int]bool)
	for _, pm := range a.processManagers {
		status := pm.GetStatus()
		if status.Status == "running" {
			// 假设端口存储在进程管理器中
			usedPorts[7400] = true // 简化处理
		}
	}

	// 从 7400 开始寻找可用端口
	for port := 7400; port < 7500; port++ {
		if !usedPorts[port] {
			return port
		}
	}

	return 7400 // 默认端口
}

// FrpcStop 停止 frpc 进程
func (a *App) FrpcStop(configID string) error {
	a.processMu.Lock()
	defer a.processMu.Unlock()

	pm, exists := a.processManagers[configID]
	if !exists {
		return nil
	}

	return pm.Stop()
}

// FrpcRestart 重启 frpc 进程
func (a *App) FrpcRestart(configID string) error {
	if a.configManager == nil {
		return fmt.Errorf("配置管理器未初始化")
	}
	if a.versionManager == nil {
		return fmt.Errorf("版本管理器未初始化")
	}

	// 获取配置
	cfg, err := a.configManager.Get(configID)
	if err != nil {
		return err
	}
	if cfg == nil {
		return fmt.Errorf("配置不存在: %s", configID)
	}

	// 获取活动版本的 frpc 路径
	frpcPath, err := a.versionManager.GetActiveFrpcPath()
	if err != nil {
		return err
	}

	// 序列化配置为 TOML 格式
	configContent, err := a.configManager.SerializeConfig(cfg, "toml")
	if err != nil {
		return err
	}

	a.processMu.Lock()
	defer a.processMu.Unlock()

	pm, exists := a.processManagers[configID]
	if !exists {
		pm = frpc.NewProcessManager()
		pm.SetFrpcPath(frpcPath)
		a.processManagers[configID] = pm
	}

	return pm.Restart(a.ctx, configID, string(configContent))
}

// FrpcGetStatus 获取进程状态
func (a *App) FrpcGetStatus(configID string) *types.ProcessState {
	a.processMu.RLock()
	defer a.processMu.RUnlock()

	pm, exists := a.processManagers[configID]
	if !exists {
		return &types.ProcessState{
			ConfigID: configID,
			Status:   "stopped",
		}
	}

	return pm.GetStatus()
}

// FrpcGetLogs 获取进程日志
func (a *App) FrpcGetLogs(configID string, limit int) []*types.LogEntry {
	a.processMu.RLock()
	defer a.processMu.RUnlock()

	pm, exists := a.processManagers[configID]
	if !exists {
		return []*types.LogEntry{}
	}

	return pm.GetLogs(limit)
}

// FrpcGetAllStatus 获取所有进程状态
func (a *App) FrpcGetAllStatus() map[string]*types.ProcessState {
	a.processMu.RLock()
	defer a.processMu.RUnlock()

	result := make(map[string]*types.ProcessState)
	for id, pm := range a.processManagers {
		result[id] = pm.GetStatus()
	}

	return result
}

// FrpcClearLogs 清除进程日志
func (a *App) FrpcClearLogs(configID string) {
	a.processMu.Lock()
	defer a.processMu.Unlock()

	pm, exists := a.processManagers[configID]
	if exists {
		pm.ClearLogs()
	}
}

// ==================== 应用日志 API ====================

// addAppLog 添加应用日志
func (a *App) addAppLog(level, message string) {
	a.appLogsMu.Lock()
	defer a.appLogsMu.Unlock()

	entry := &types.LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	}

	// 保留最近 1000 条日志
	if len(a.appLogs) >= 1000 {
		a.appLogs = a.appLogs[1:]
	}

	a.appLogs = append(a.appLogs, entry)
}

// AppGetLogs 获取应用日志
func (a *App) AppGetLogs(limit int) []*types.LogEntry {
	a.appLogsMu.RLock()
	defer a.appLogsMu.RUnlock()

	if limit <= 0 || limit > len(a.appLogs) {
		limit = len(a.appLogs)
	}

	// 返回最新的 limit 条日志
	start := len(a.appLogs) - limit
	if start < 0 {
		start = 0
	}

	result := make([]*types.LogEntry, limit)
	copy(result, a.appLogs[start:])
	return result
}

// AppClearLogs 清除应用日志
func (a *App) AppClearLogs() {
	a.appLogsMu.Lock()
	defer a.appLogsMu.Unlock()

	a.appLogs = make([]*types.LogEntry, 0)
	a.addAppLog("info", "日志已清除")
}

// ==================== 开机启动 API ====================

// InitStartupManager 初始化开机启动管理器
func (a *App) InitStartupManager(manager *startup.Manager) {
	a.startupManager = manager
	// 检查当前设置状态，如果已启用则确保注册表正确
	if a.sqliteStore != nil {
		autoStart, _ := a.sqliteStore.GetSetting("autoStart")
		if autoStart == "true" && manager != nil {
			manager.Enable()
		}
	}
}

// SetAutoStart 设置开机启动
func (a *App) SetAutoStart(enabled bool) error {
	if a.startupManager == nil {
		return fmt.Errorf("开机启动管理器未初始化")
	}

	if enabled {
		return a.startupManager.Enable()
	} else {
		return a.startupManager.Disable()
	}
}

// GetAutoStart 获取开机启动状态
func (a *App) GetAutoStart() bool {
	if a.startupManager == nil {
		return false
	}
	return a.startupManager.IsEnabled()
}

// autoStartMarkedConfigs 自动启动标记为 AutoStartOnLaunch 的配置
func (a *App) autoStartMarkedConfigs() {
	// 等待版本管理器初始化完成（延迟启动，避免阻塞主流程）
	time.Sleep(2 * time.Second)

	// 检查是否有活动版本
	if a.versionManager == nil || a.versionManager.GetActive() == "" {
		log.Println("[AutoStart] 无活动版本，跳过自动启动")
		return
	}

	// 获取所有配置
	configs, err := a.configManager.List()
	if err != nil {
		log.Printf("[AutoStart] 获取配置列表失败: %v", err)
		return
	}

	// 遍历并启动标记的配置
	for _, meta := range configs {
		if meta.AutoStartOnLaunch {
			log.Printf("[AutoStart] 自动启动配置: %s (%s)", meta.Name, meta.ID)
			if err := a.FrpcStart(meta.ID); err != nil {
				log.Printf("[AutoStart] 启动配置 %s 失败: %v", meta.Name, err)
			} else {
				a.addAppLog("info", fmt.Sprintf("自动启动配置: %s", meta.Name))
			}
		}
	}
}
