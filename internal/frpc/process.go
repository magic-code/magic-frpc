// Package frpc 提供 frpc 进程管理功能
package frpc

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/magic-frpc/gui/internal/dashboard"
	"github.com/magic-frpc/gui/pkg/types"
)

// ProcessStatus 进程状态常量
const (
	StatusStopped = "stopped"
	StatusRunning = "running"
	StatusError   = "error"
)

// EventCallback 事件回调函数类型
type EventCallback func(eventType string, data interface{})

// ProcessManager 进程管理器
type ProcessManager struct {
	mu sync.RWMutex

	// 进程命令
	cmd *exec.Cmd

	// 进程状态
	state *types.ProcessState

	// 配置文件路径
	configPath string

	// frpc 二进制路径
	frpcPath string

	// Dashboard 配置
	dashboardAddr string
	dashboardPort int
	dashboard     *dashboard.Client

	// 日志缓冲区
	logBuffer []*types.LogEntry
	logMu     sync.Mutex
	maxLogs   int

	// 上一次流量采样值（用于计算速率）
	lastTotalIn    int64
	lastTotalOut   int64
	lastSampleTime time.Time

	// 事件回调
	callback EventCallback

	// 上下文和取消函数
	ctx    context.Context
	cancel context.CancelFunc

	// 输出管道
	stdoutPipe io.ReadCloser
	stderrPipe io.ReadCloser
}

// NewProcessManager 创建进程管理器
func NewProcessManager() *ProcessManager {
	return &ProcessManager{
		state: &types.ProcessState{
			Status: StatusStopped,
		},
		logBuffer: make([]*types.LogEntry, 0),
		maxLogs:   1000,
	}
}

// SetCallback 设置事件回调
func (pm *ProcessManager) SetCallback(cb EventCallback) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.callback = cb
}

// SetFrpcPath 设置 frpc 二进制路径
func (pm *ProcessManager) SetFrpcPath(path string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.frpcPath = path
}

// SetDashboard 设置 Dashboard 配置（地址、端口、用户名、密码）
func (pm *ProcessManager) SetDashboard(addr string, port int, username, password string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.dashboardAddr = addr
	pm.dashboardPort = port
	if port > 0 {
		pm.dashboard = dashboard.NewClient(addr, port, username, password)
	} else {
		pm.dashboard = nil
	}
}

// Start 启动 frpc 进程
func (pm *ProcessManager) Start(ctx context.Context, configID string, configContent string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 检查是否已在运行
	if pm.state.Status == StatusRunning {
		return fmt.Errorf("进程已在运行中")
	}

	// 检查 frpc 路径
	if pm.frpcPath == "" {
		return fmt.Errorf("未设置 frpc 二进制路径")
	}

	// 检查 frpc 文件是否存在
	if _, err := os.Stat(pm.frpcPath); os.IsNotExist(err) {
		return fmt.Errorf("frpc 二进制文件不存在: %s", pm.frpcPath)
	}

	// 创建临时配置文件
	tempDir := os.TempDir()
	configPath := filepath.Join(tempDir, fmt.Sprintf("frpc_%s.toml", configID))

	// 打印生成的配置内容用于调试
	log.Printf("========== 生成的 frpc 配置 ==========")
	log.Printf("%s", configContent)
	log.Printf("======================================")

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("创建配置文件失败: %v", err)
	}
	pm.configPath = configPath

	log.Printf("配置文件保存到: %s", configPath)

	// 创建子上下文
	pm.ctx, pm.cancel = context.WithCancel(ctx)

	// 创建命令
	pm.cmd = exec.CommandContext(pm.ctx, pm.frpcPath, "-c", configPath)

	// 创建管道
	var err error
	pm.stdoutPipe, err = pm.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("创建 stdout 管道失败: %v", err)
	}
	pm.stderrPipe, err = pm.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("创建 stderr 管道失败: %v", err)
	}

	// 启动进程
	if err := pm.cmd.Start(); err != nil {
		return fmt.Errorf("启动进程失败: %v", err)
	}

	// 保存已有的代理列表
	existingProxies := pm.state.Proxies

	// 更新状态
	pm.state = &types.ProcessState{
		ConfigID:  configID,
		Status:    StatusRunning,
		PID:       pm.cmd.Process.Pid,
		StartedAt: time.Now(),
		Proxies:   existingProxies, // 保留之前设置的代理列表
	}

	log.Printf("[Start] 进程启动，代理列表: %v", pm.state.Proxies)

	// 启动输出捕获协程
	go pm.captureOutput(pm.stdoutPipe, "info")
	go pm.captureOutput(pm.stderrPipe, "error")

	// 启动进程监控协程
	go pm.monitorProcess()

	// 启动流量统计协程（如果 Dashboard 已配置）
	if pm.dashboard != nil {
		go pm.collectTrafficStats()
	}

	log.Printf("frpc 进程已启动, PID: %d", pm.cmd.Process.Pid)
	pm.emitEvent("started", pm.state)

	return nil
}

// Stop 停止 frpc 进程
func (pm *ProcessManager) Stop() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.cmd == nil || pm.cmd.Process == nil {
		return nil
	}

	if pm.state.Status != StatusRunning {
		return nil
	}

	log.Printf("正在停止 frpc 进程, PID: %d", pm.cmd.Process.Pid)

	// 尝试优雅终止（Windows 上直接 Kill）
	if err := pm.cmd.Process.Kill(); err != nil {
		// 即使 Kill 失败，也要更新状态
		pm.state.Status = StatusStopped
		pm.state.StoppedAt = time.Now()
		pm.cleanup()
		return fmt.Errorf("终止进程失败: %v", err)
	}

	// 等待进程退出（最多 5 秒）
	done := make(chan error, 1)
	go func() {
		done <- pm.cmd.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		log.Printf("等待进程退出超时")
		// 超时也要更新状态
		pm.state.Status = StatusStopped
		pm.state.StoppedAt = time.Now()
		pm.cleanup()
	case err := <-done:
		if err != nil {
			log.Printf("进程退出错误: %v", err)
			if exitErr, ok := err.(*exec.ExitError); ok {
				pm.state.ExitCode = exitErr.ExitCode()
				pm.state.ExitMessage = exitErr.Error()
			}
		}
		// 更新状态为已停止
		pm.state.Status = StatusStopped
		pm.state.StoppedAt = time.Now()
		pm.cleanup()
	}

	log.Printf("frpc 进程已停止")
	pm.emitEvent("stopped", pm.state)

	return nil
}

// Restart 重启 frpc 进程
func (pm *ProcessManager) Restart(ctx context.Context, configID string, configContent string) error {
	// 先停止
	if err := pm.Stop(); err != nil {
		log.Printf("停止进程时出错: %v", err)
	}

	// 等待一小段时间
	time.Sleep(500 * time.Millisecond)

	// 重新启动
	return pm.Start(ctx, configID, configContent)
}

// GetStatus 获取进程状态
func (pm *ProcessManager) GetStatus() *types.ProcessState {
	pm.mu.RLock()
	state := *pm.state
	pm.mu.RUnlock()

	return &state
}

// SetProxies 设置代理列表（从配置中获取）
func (pm *ProcessManager) SetProxies(proxies []types.ProxyStatus) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.state != nil {
		pm.state.Proxies = proxies
	}
}

// UpdateProxyStatus 更新单个代理状态
func (pm *ProcessManager) UpdateProxyStatus(proxyName string, status string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.state == nil {
		log.Printf("[UpdateProxyStatus] state 为 nil")
		return
	}
	log.Printf("[UpdateProxyStatus] 尝试更新代理 '%s' 为 '%s', 当前代理列表: %v", proxyName, status, pm.state.Proxies)
	for i := range pm.state.Proxies {
		log.Printf("[UpdateProxyStatus] 检查代理[%d]: '%s'", i, pm.state.Proxies[i].Name)
		if pm.state.Proxies[i].Name == proxyName {
			pm.state.Proxies[i].Status = status
			log.Printf("[UpdateProxyStatus] 成功更新代理 '%s' 状态为 '%s'", proxyName, status)
			break
		}
	}
}

// GetLogs 获取日志
func (pm *ProcessManager) GetLogs(limit int) []*types.LogEntry {
	pm.logMu.Lock()
	defer pm.logMu.Unlock()

	if limit <= 0 || limit > len(pm.logBuffer) {
		limit = len(pm.logBuffer)
	}

	// 返回最近的日志
	start := len(pm.logBuffer) - limit
	if start < 0 {
		start = 0
	}

	logs := make([]*types.LogEntry, limit)
	copy(logs, pm.logBuffer[start:])
	return logs
}

// ClearLogs 清除日志
func (pm *ProcessManager) ClearLogs() {
	pm.logMu.Lock()
	defer pm.logMu.Unlock()
	pm.logBuffer = make([]*types.LogEntry, 0)
}

// captureOutput 捕获进程输出
func (pm *ProcessManager) captureOutput(pipe io.Reader, level string) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		entry := &types.LogEntry{
			Timestamp: time.Now(),
			Level:     level,
			Message:   line,
		}

		// 添加到缓冲区
		pm.logMu.Lock()
		pm.logBuffer = append(pm.logBuffer, entry)
		// 限制日志数量
		if len(pm.logBuffer) > pm.maxLogs {
			pm.logBuffer = pm.logBuffer[1:]
		}
		pm.logMu.Unlock()

		// 解析日志更新代理状态
		pm.parseLogForProxyStatus(line)

		// 发送日志事件
		pm.emitEvent("log", entry)
	}
}

// parseLogForProxyStatus 从日志中解析代理状态
func (pm *ProcessManager) parseLogForProxyStatus(line string) {
	// frpc 日志格式示例：
	// [I] [proxy_name] start proxy success
	// [I] [proxy.go:162] [proxy_name] start proxy success
	// [I] [client/control.go:176] [runid] [proxy_name] start proxy success
	// [E] [proxy_name] start proxy error: ...

	// 调试：打印原始日志
	log.Printf("[frpc日志] %s", line)

	// 检测代理启动成功
	if containsAll(line, "start proxy success") {
		proxyName := extractProxyName(line, "start proxy success")
		log.Printf("[解析] 检测到代理启动成功，提取名称: '%s'", proxyName)
		if proxyName != "" {
			pm.UpdateProxyStatus(proxyName, "running")
			log.Printf("[Proxy] %s 启动成功", proxyName)
		}
	}

	// 检测代理启动失败
	if containsAll(line, "start proxy error") || containsAll(line, "start proxy failed") {
		proxyName := extractProxyName(line, "start proxy")
		log.Printf("[解析] 检测到代理启动失败，提取名称: '%s'", proxyName)
		if proxyName != "" {
			pm.UpdateProxyStatus(proxyName, "error")
			log.Printf("[Proxy] %s 启动失败", proxyName)
		}
	}

	// 检测连接成功
	if containsAll(line, "login to server success") {
		log.Printf("[Proxy] 已成功连接到服务器")
	}

	// 检测连接失败
	if containsAll(line, "login to server failed") || containsAll(line, "connect to server error") {
		log.Printf("[Proxy] 连接服务器失败")
	}
}

// stripANSI 移除 ANSI 颜色代码
func stripANSI(s string) string {
	// 移除 ANSI 转义序列，如 [0m, [1;34m 等
	result := ""
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			inEscape = true
			continue
		}
		if inEscape {
			if s[i] == 'm' {
				inEscape = false
			}
			continue
		}
		result += string(s[i])
	}
	return result
}

// containsAll 检查字符串是否包含所有子串
func containsAll(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if !strings.Contains(strings.ToLower(s), strings.ToLower(substr)) {
			return false
		}
	}
	return true
}

// extractProxyName 从日志行中提取代理名称
func extractProxyName(line string, marker string) string {
	// 先移除 ANSI 颜色代码
	cleanLine := stripANSI(line)

	// 查找 marker 的位置
	idx := strings.Index(cleanLine, marker)
	if idx == -1 {
		return ""
	}

	// 向前查找最近的 [xxx] 格式的代理名称
	before := cleanLine[:idx]
	lastOpen := strings.LastIndex(before, "[")
	if lastOpen == -1 {
		return ""
	}

	lastClose := strings.Index(before[lastOpen:], "]")
	if lastClose == -1 {
		return ""
	}

	return strings.TrimSpace(before[lastOpen+1 : lastOpen+lastClose])
}

// monitorProcess 监控进程状态
func (pm *ProcessManager) monitorProcess() {
	if pm.cmd == nil {
		return
	}

	err := pm.cmd.Wait()

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 如果状态已经不是 running（可能被 Stop() 更新了），则跳过
	if pm.state.Status != StatusRunning {
		return
	}

	// 更新状态
	pm.state.Status = StatusStopped
	pm.state.StoppedAt = time.Now()

	if err != nil {
		pm.state.Status = StatusError
		if exitErr, ok := err.(*exec.ExitError); ok {
			pm.state.ExitCode = exitErr.ExitCode()
			pm.state.ExitMessage = exitErr.Error()
		}
		log.Printf("进程异常退出: %v", err)
	} else {
		pm.state.ExitCode = 0
	}

	// 清理
	pm.cleanup()

	// 发送退出事件
	pm.emitEvent("exited", pm.state)
}

// collectTrafficStats 定时从 Dashboard 获取流量统计
func (pm *ProcessManager) collectTrafficStats() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	log.Printf("[TrafficStats] 开始流量统计收集，Dashboard: %s:%d", pm.dashboardAddr, pm.dashboardPort)

	for {
		select {
		case <-pm.ctx.Done():
			log.Printf("[TrafficStats] 上下文已取消，停止收集")
			return
		case <-ticker.C:
			if pm.dashboard == nil {
				log.Printf("[TrafficStats] Dashboard 客户端为 nil，跳过")
				return
			}

			// 获取当前代理名称列表
			pm.mu.RLock()
			proxyNames := make([]string, 0, len(pm.state.Proxies))
			for _, p := range pm.state.Proxies {
				proxyNames = append(proxyNames, p.Name)
			}
			pm.mu.RUnlock()

			if len(proxyNames) == 0 {
				log.Printf("[TrafficStats] 没有代理列表，跳过")
				continue
			}

			// 使用 /api/traffic/{name} 接口获取每个代理的流量
			proxies, totalIn, totalOut, err := pm.dashboard.GetProxyStatusesWithNames(proxyNames)
			if err != nil {
				log.Printf("[TrafficStats] 获取流量失败: %v", err)
				continue
			}

			// 没有代理数据则跳过
			if len(proxies) == 0 {
				log.Printf("[TrafficStats] 未获取到代理数据")
				continue
			}

			log.Printf("[TrafficStats] 获取到 %d 个代理，总流量: 入=%d, 出=%d", len(proxies), totalIn, totalOut)

			now := time.Now()

			pm.mu.Lock()
			if pm.state != nil && pm.state.Status == StatusRunning {
				// 计算总速率
				if !pm.lastSampleTime.IsZero() {
					elapsed := now.Sub(pm.lastSampleTime).Seconds()
					if elapsed > 0 {
						pm.state.CurrentSpeedIn = int64(float64(totalIn-pm.lastTotalIn) / elapsed)
						pm.state.CurrentSpeedOut = int64(float64(totalOut-pm.lastTotalOut) / elapsed)
					}
				}

				// 更新总流量
				pm.state.TotalTrafficIn = totalIn
				pm.state.TotalTrafficOut = totalOut

				// 更新每个代理的流量和速率
				for _, proxy := range proxies {
					for i := range pm.state.Proxies {
						if pm.state.Proxies[i].Name == proxy.Name {
							// 计算代理速率
							if !pm.lastSampleTime.IsZero() {
								elapsed := now.Sub(pm.lastSampleTime).Seconds()
								if elapsed > 0 {
									pm.state.Proxies[i].CurrentSpeedIn = int64(float64(proxy.TodayTrafficIn-pm.state.Proxies[i].TodayTrafficIn) / elapsed)
									pm.state.Proxies[i].CurrentSpeedOut = int64(float64(proxy.TodayTrafficOut-pm.state.Proxies[i].TodayTrafficOut) / elapsed)
								}
							}
							pm.state.Proxies[i].TodayTrafficIn = proxy.TodayTrafficIn
							pm.state.Proxies[i].TodayTrafficOut = proxy.TodayTrafficOut
							pm.state.Proxies[i].LocalAddr = proxy.LocalAddr
							pm.state.Proxies[i].RemoteAddr = proxy.RemoteAddr
							// 如果代理状态未知，使用 Dashboard 返回的状态
							if pm.state.Proxies[i].Status == "starting" {
								pm.state.Proxies[i].Status = proxy.Status
							}
							break
						}
					}
				}

				// 保存本次采样值，供下次计算速率
				pm.lastTotalIn = totalIn
				pm.lastTotalOut = totalOut
				pm.lastSampleTime = now
			}
			pm.mu.Unlock()
		}
	}
}

// resetTrafficState 重置流量统计状态
func (pm *ProcessManager) resetTrafficState() {
	if pm.state != nil {
		pm.state.CurrentSpeedIn = 0
		pm.state.CurrentSpeedOut = 0
	}
	pm.lastTotalIn = 0
	pm.lastTotalOut = 0
	pm.lastSampleTime = time.Time{}
}

// cleanup 清理资源（可以安全地多次调用）
func (pm *ProcessManager) cleanup() {
	// 重置流量统计
	pm.resetTrafficState()

	// 关闭管道（安全关闭）
	if pm.stdoutPipe != nil {
		pm.stdoutPipe.Close()
		pm.stdoutPipe = nil
	}
	if pm.stderrPipe != nil {
		pm.stderrPipe.Close()
		pm.stderrPipe = nil
	}

	// 删除临时配置文件
	if pm.configPath != "" {
		if err := os.Remove(pm.configPath); err != nil && !os.IsNotExist(err) {
			log.Printf("删除临时配置文件失败: %v", err)
		}
		pm.configPath = ""
	}

	// 取消上下文
	if pm.cancel != nil {
		pm.cancel()
		pm.cancel = nil
	}

	pm.cmd = nil
}

// emitEvent 发送事件
func (pm *ProcessManager) emitEvent(eventType string, data interface{}) {
	if pm.callback != nil {
		go pm.callback(eventType, data)
	}
}

// IsRunning 检查进程是否在运行
func (pm *ProcessManager) IsRunning() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.state.Status == StatusRunning
}

// GetPID 获取进程 PID
func (pm *ProcessManager) GetPID() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.state.PID
}
