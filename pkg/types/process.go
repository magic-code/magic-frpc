// Package types 定义进程状态类型
package types

import "time"

// ProcessState 进程状态
type ProcessState struct {
	ConfigID    string    `json:"configId"`
	Status      string    `json:"status"` // running, stopped, error
	PID         int       `json:"pid"`
	StartedAt   time.Time `json:"startedAt,omitempty"`
	StoppedAt   time.Time `json:"stoppedAt,omitempty"`
	ExitCode    int       `json:"exitCode,omitempty"`
	ExitMessage string    `json:"exitMessage,omitempty"`

	// 代理状态信息（新增）
	Proxies         []ProxyStatus `json:"proxies,omitempty"`
	TotalTrafficIn  int64         `json:"totalTrafficIn,omitempty"`  // 总入流量(字节)
	TotalTrafficOut int64         `json:"totalTrafficOut,omitempty"` // 总出流量(字节)
	CurrentSpeedIn  int64         `json:"currentSpeedIn,omitempty"`  // 当前入速度(字节/秒)
	CurrentSpeedOut int64         `json:"currentSpeedOut,omitempty"` // 当前出速度(字节/秒)
}

// ProxyStatus 代理状态
type ProxyStatus struct {
	Name            string `json:"name"`
	Type            string `json:"type"`
	LocalAddr       string `json:"localAddr"`
	RemoteAddr      string `json:"remoteAddr"`
	Status          string `json:"status"`          // running, offline, error
	TodayTrafficIn  int64  `json:"todayTrafficIn"`  // 今日入流量(字节)
	TodayTrafficOut int64  `json:"todayTrafficOut"` // 今日出流量(字节)
	CurrentSpeedIn  int64  `json:"currentSpeedIn"`  // 当前入速度(字节/秒)
	CurrentSpeedOut int64  `json:"currentSpeedOut"` // 当前出速度(字节/秒)
}

// ProcessInfo 进程信息
type ProcessInfo struct {
	ConfigID string `json:"configId"`
	PID      int    `json:"pid"`
	Status   string `json:"status"`
}

// LogEntry 日志条目
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}
