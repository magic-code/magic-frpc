// Package config 提供配置验证功能
package config

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/magic-frpc/gui/pkg/types"
)

// Validator 配置验证器
type Validator struct{}

// NewValidator 创建验证器
func NewValidator() *Validator {
	return &Validator{}
}

// ValidationResult 验证结果
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// Validate 验证配置
func (v *Validator) Validate(config *types.FrpcConfig) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// 验证配置名称
	if strings.TrimSpace(config.Name) == "" {
		result.Errors = append(result.Errors, "配置名称不能为空")
		result.Valid = false
	}

	// 验证 common 配置
	// 注意：允许在没有 common 配置的情况下保存（用户可以稍后配置）
	if config.Common == nil {
		result.Warnings = append(result.Warnings, "缺少 [common] 配置节，启动 frpc 前需要配置服务器信息")
	} else {
		v.validateCommon(config.Common, result)
	}

	// 验证代理规则
	if len(config.Proxies) == 0 {
		result.Warnings = append(result.Warnings, "未配置任何代理规则")
	} else {
		v.validateProxies(config.Proxies, result)
	}

	return result
}

// validateCommon 验证 common 配置
func (v *Validator) validateCommon(common *types.CommonSection, result *ValidationResult) {
	// 验证服务器地址
	if common.ServerAddr == "" {
		result.Errors = append(result.Errors, "服务器地址不能为空")
		result.Valid = false
	} else if !v.isValidHost(common.ServerAddr) {
		result.Warnings = append(result.Warnings, "服务器地址格式可能不正确")
	}

	// 验证服务器端口
	if common.ServerPort <= 0 || common.ServerPort > 65535 {
		result.Errors = append(result.Errors, "服务器端口必须在 1-65535 范围内")
		result.Valid = false
	}

	// 验证协议
	validProtocols := map[string]bool{"tcp": true, "kcp": true, "websocket": true, "wss": true, "quic": true}
	if common.Protocol != "" && !validProtocols[strings.ToLower(common.Protocol)] {
		result.Warnings = append(result.Warnings, fmt.Sprintf("协议 '%s' 可能不被支持", common.Protocol))
	}

	// 认证方式检查
	if common.AuthToken != "" && common.AuthMethod == "" {
		result.Warnings = append(result.Warnings, "设置了认证令牌但未指定认证方式")
	}
}

// validateProxies 验证代理配置
func (v *Validator) validateProxies(proxies []types.ProxyConfig, result *ValidationResult) {
	// 检查代理名称是否重复
	names := make(map[string]int)
	for i, proxy := range proxies {
		// 检查名称
		if proxy.Name == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("代理 #%d: 名称不能为空", i+1))
			result.Valid = false
			continue
		}

		names[proxy.Name]++
		if names[proxy.Name] > 1 {
			result.Errors = append(result.Errors, fmt.Sprintf("代理名称 '%s' 重复", proxy.Name))
			result.Valid = false
		}

		// 检查类型
		validTypes := map[string]bool{
			"tcp": true, "udp": true, "http": true, "https": true,
			"stcp": true, "sudp": true, "xtcp": true, "xudp": true, "tcpmux": true,
		}
		if proxy.Type == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("代理 '%s': 类型不能为空", proxy.Name))
			result.Valid = false
		} else if !validTypes[strings.ToLower(proxy.Type)] {
			result.Warnings = append(result.Warnings, fmt.Sprintf("代理 '%s': 类型 '%s' 可能不被支持", proxy.Name, proxy.Type))
		}

		// 检查本地 IP
		if proxy.LocalIP == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("代理 '%s': 本地 IP 为空，将使用默认值 127.0.0.1", proxy.Name))
		} else if !v.isValidIP(proxy.LocalIP) {
			result.Warnings = append(result.Warnings, fmt.Sprintf("代理 '%s': 本地 IP '%s' 格式可能不正确", proxy.Name, proxy.LocalIP))
		}

		// 检查本地端口
		if proxy.LocalPort <= 0 || proxy.LocalPort > 65535 {
			result.Errors = append(result.Errors, fmt.Sprintf("代理 '%s': 本地端口必须在 1-65535 范围内", proxy.Name))
			result.Valid = false
		}

		// 根据类型检查必要字段
		switch strings.ToLower(proxy.Type) {
		case "tcp", "udp":
			if proxy.RemotePort <= 0 {
				result.Warnings = append(result.Warnings, fmt.Sprintf("代理 '%s': 类型 %s 通常需要指定远程端口", proxy.Name, proxy.Type))
			}
		case "http", "https":
			if len(proxy.CustomDomains) == 0 && proxy.Subdomain == "" {
				result.Warnings = append(result.Warnings, fmt.Sprintf("代理 '%s': 类型 %s 需要配置域名或子域名", proxy.Name, proxy.Type))
			}
		case "stcp", "sudp", "xtcp", "xudp":
			if proxy.RemotePort <= 0 {
				result.Warnings = append(result.Warnings, fmt.Sprintf("代理 '%s': 类型 %s 通常需要指定远程端口", proxy.Name, proxy.Type))
			}
			if proxy.Sk == "" {
				result.Warnings = append(result.Warnings, fmt.Sprintf("代理 '%s': 类型 %s 建议设置访问密钥 (sk)", proxy.Name, proxy.Type))
			}
		}
	}
}

// isValidHost 检查主机地址是否有效
func (v *Validator) isValidHost(host string) bool {
	// 检查是否是 IP 地址
	if v.isValidIP(host) {
		return true
	}

	// 检查是否是有效域名
	if match, _ := regexp.MatchString(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`, host); match {
		return true
	}

	return false
}

// isValidIP 检查 IP 地址是否有效
func (v *Validator) isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// ValidateServerAddr 验证服务器地址格式
func (v *Validator) ValidateServerAddr(addr string) bool {
	return v.isValidHost(addr)
}

// ValidatePort 验证端口格式
func (v *Validator) ValidatePort(port string) bool {
	p, err := strconv.Atoi(port)
	if err != nil {
		return false
	}
	return p > 0 && p <= 65535
}
