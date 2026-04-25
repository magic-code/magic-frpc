// Package config 提供配置解析器
package config

import (
	"bytes"
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/magic-frpc/gui/pkg/types"
)

// TOMLParser TOML 配置解析器
type TOMLParser struct{}

// NewTOMLParser 创建 TOML 解析器
func NewTOMLParser() *TOMLParser {
	return &TOMLParser{}
}

// Parse 解析 TOML 配置
func (p *TOMLParser) Parse(data []byte) (*types.FrpcConfig, error) {
	var raw map[string]interface{}
	if _, err := toml.Decode(string(data), &raw); err != nil {
		return nil, fmt.Errorf("解析 TOML 失败: %w", err)
	}

	config := &types.FrpcConfig{
		Tags:    []string{},
		Proxies: []types.ProxyConfig{},
	}

	// 解析 [common] 节
	if common, ok := raw["common"].(map[string]interface{}); ok {
		config.Common = &types.CommonSection{}
		if v, ok := common["server_addr"].(string); ok {
			config.Common.ServerAddr = v
		}
		if v, ok := common["server_port"].(int64); ok {
			config.Common.ServerPort = int(v)
		}
		if v, ok := common["auth_token"].(string); ok {
			config.Common.AuthToken = v
		}
		if v, ok := common["auth_method"].(string); ok {
			config.Common.AuthMethod = v
		}
		if v, ok := common["user"].(string); ok {
			config.Common.User = v
		}
		if v, ok := common["protocol"].(string); ok {
			config.Common.Protocol = v
		}
		if v, ok := common["tls_enable"].(bool); ok {
			config.Common.TLSEnable = v
		}
		if v, ok := common["server_name"].(string); ok {
			config.Common.ServerName = v
		}
		if v, ok := common["heartbeat_interval"].(int64); ok {
			config.Common.HeartbeatInterval = int(v)
		}
		if v, ok := common["heartbeat_timeout"].(int64); ok {
			config.Common.HeartbeatTimeout = int(v)
		}
		// Dashboard 配置
		if v, ok := common["admin_addr"].(string); ok {
			config.Common.AdminAddr = v
		}
		if v, ok := common["admin_port"].(int64); ok {
			config.Common.AdminPort = int(v)
		}
	}

	// 解析代理规则 [[proxies]]
	for key, value := range raw {
		// 跳过 common 节
		if key == "common" {
			continue
		}

		// 检查是否是代理配置
		if proxyMap, ok := value.(map[string]interface{}); ok {
			proxy := types.ProxyConfig{
				Name: key,
			}

			if v, ok := proxyMap["type"].(string); ok {
				proxy.Type = v
			}
			if v, ok := proxyMap["local_ip"].(string); ok {
				proxy.LocalIP = v
			}
			if v, ok := proxyMap["local_port"].(int64); ok {
				proxy.LocalPort = int(v)
			}
			if v, ok := proxyMap["remote_port"].(int64); ok {
				proxy.RemotePort = int(v)
			}
			if v, ok := proxyMap["subdomain"].(string); ok {
				proxy.Subdomain = v
			}
			if v, ok := proxyMap["sk"].(string); ok {
				proxy.Sk = v
			}
			if v, ok := proxyMap["host_header_rewrite"].(string); ok {
				proxy.HostHeaderRewrite = v
			}
			if v, ok := proxyMap["custom_domains"].([]interface{}); ok {
				for _, d := range v {
					if domain, ok := d.(string); ok {
						proxy.CustomDomains = append(proxy.CustomDomains, domain)
					}
				}
			}

			config.Proxies = append(config.Proxies, proxy)
		}
	}

	return config, nil
}

// Serialize 序列化为 TOML 格式 (frpc v0.52+ 新格式)
func (p *TOMLParser) Serialize(config *types.FrpcConfig) ([]byte, error) {
	var buf bytes.Buffer

	// 写入服务器配置（新版格式使用 camelCase）
	if config.Common != nil {
		if config.Common.ServerAddr != "" {
			buf.WriteString(fmt.Sprintf("serverAddr = %q\n", config.Common.ServerAddr))
		}
		if config.Common.ServerPort > 0 {
			buf.WriteString(fmt.Sprintf("serverPort = %d\n", config.Common.ServerPort))
		}
		if config.Common.AuthToken != "" {
			buf.WriteString(fmt.Sprintf("auth.token = %q\n", config.Common.AuthToken))
		}
		if config.Common.AuthMethod != "" {
			buf.WriteString(fmt.Sprintf("auth.method = %q\n", config.Common.AuthMethod))
		}
		if config.Common.User != "" {
			buf.WriteString(fmt.Sprintf("user = %q\n", config.Common.User))
		}
		if config.Common.Protocol != "" && config.Common.Protocol != "tcp" {
			buf.WriteString(fmt.Sprintf("transport.protocol = %q\n", config.Common.Protocol))
		}
		if config.Common.TLSEnable {
			buf.WriteString("transport.tls.enable = true\n")
		}
		if config.Common.ServerName != "" {
			buf.WriteString(fmt.Sprintf("transport.tls.serverName = %q\n", config.Common.ServerName))
		}
		if config.Common.HeartbeatInterval > 0 && config.Common.HeartbeatInterval != 10 {
			buf.WriteString(fmt.Sprintf("transport.heartbeatInterval = %d\n", config.Common.HeartbeatInterval))
		}
		if config.Common.HeartbeatTimeout > 0 && config.Common.HeartbeatTimeout != 90 {
			buf.WriteString(fmt.Sprintf("transport.heartbeatTimeout = %d\n", config.Common.HeartbeatTimeout))
		}
		// Dashboard 配置 - frpc 本地绑定地址始终使用 127.0.0.1
		// AdminAddr 仅用于 GUI 请求远程 Dashboard API，不写入配置文件
		if config.Common.AdminPort > 0 {
			buf.WriteString("webServer.addr = \"127.0.0.1\"\n")
			buf.WriteString(fmt.Sprintf("webServer.port = %d\n", config.Common.AdminPort))
			buf.WriteString("webServer.user = \"admin\"\n")
			buf.WriteString("webServer.password = \"admin\"\n")
		}
		buf.WriteString("\n")
	}

	// 写入代理规则（新版格式使用 [[proxies]]）
	for _, proxy := range config.Proxies {
		buf.WriteString("[[proxies]]\n")
		if proxy.Name != "" {
			buf.WriteString(fmt.Sprintf("name = %q\n", proxy.Name))
		}
		if proxy.Type != "" {
			buf.WriteString(fmt.Sprintf("type = %q\n", proxy.Type))
		}
		if proxy.LocalIP != "" {
			buf.WriteString(fmt.Sprintf("localIP = %q\n", proxy.LocalIP))
		}
		if proxy.LocalPort > 0 {
			buf.WriteString(fmt.Sprintf("localPort = %d\n", proxy.LocalPort))
		}
		if proxy.RemotePort > 0 {
			buf.WriteString(fmt.Sprintf("remotePort = %d\n", proxy.RemotePort))
		}
		if proxy.Subdomain != "" {
			buf.WriteString(fmt.Sprintf("subdomain = %q\n", proxy.Subdomain))
		}
		if proxy.Sk != "" {
			buf.WriteString(fmt.Sprintf("secretKey = %q\n", proxy.Sk))
		}
		if proxy.HostHeaderRewrite != "" {
			buf.WriteString(fmt.Sprintf("hostHeaderRewrite = %q\n", proxy.HostHeaderRewrite))
		}
		if len(proxy.CustomDomains) > 0 {
			buf.WriteString("customDomains = [")
			for i, d := range proxy.CustomDomains {
				if i > 0 {
					buf.WriteString(", ")
				}
				buf.WriteString(fmt.Sprintf("%q", d))
			}
			buf.WriteString("]\n")
		}
		buf.WriteString("\n")
	}

	return buf.Bytes(), nil
}

// Extension 获取文件扩展名
func (p *TOMLParser) Extension() string {
	return ".toml"
}
