// Package config 提供配置解析器
package config

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/magic-frpc/gui/pkg/types"
	"gopkg.in/ini.v1"
)

// INIParser INI 配置解析器
type INIParser struct{}

// NewINIParser 创建 INI 解析器
func NewINIParser() *INIParser {
	return &INIParser{}
}

// Parse 解析 INI 配置
func (p *INIParser) Parse(data []byte) (*types.FrpcConfig, error) {
	cfg, err := ini.Load(data)
	if err != nil {
		return nil, fmt.Errorf("解析 INI 失败: %w", err)
	}

	config := &types.FrpcConfig{
		Tags:    []string{},
		Proxies: []types.ProxyConfig{},
	}

	// 解析 [common] 节
	if common := cfg.Section("common"); common != nil {
		config.Common = &types.CommonSection{
			ServerAddr: common.Key("server_addr").String(),
			ServerPort: p.parseInt(common.Key("server_port").String()),
			AuthToken:  common.Key("auth_token").String(),
			AuthMethod: common.Key("auth_method").String(),
			User:       common.Key("user").String(),
			Protocol:   common.Key("protocol").String(),
			TLSEnable:  common.Key("tls_enable").String() == "true",
		}
	}

	// 解析代理规则
	for _, section := range cfg.Sections() {
		name := section.Name()
		// 跳过默认节和 common 节
		if name == "DEFAULT" || name == "common" {
			continue
		}

		proxy := types.ProxyConfig{
			Name:       name,
			Type:       section.Key("type").String(),
			LocalIP:    section.Key("local_ip").String(),
			LocalPort:  p.parseInt(section.Key("local_port").String()),
			RemotePort: p.parseInt(section.Key("remote_port").String()),
			Subdomain:  section.Key("subdomain").String(),
			Sk:         section.Key("sk").String(),
		}

		// 解析 custom_domains
		if domains := section.Key("custom_domains").String(); domains != "" {
			proxy.CustomDomains = strings.Split(domains, ",")
			for i, d := range proxy.CustomDomains {
				proxy.CustomDomains[i] = strings.TrimSpace(d)
			}
		}

		config.Proxies = append(config.Proxies, proxy)
	}

	return config, nil
}

// parseInt 安全解析整数
func (p *INIParser) parseInt(s string) int {
	if s == "" {
		return 0
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return v
}

// Serialize 序列化为 INI 格式
func (p *INIParser) Serialize(config *types.FrpcConfig) ([]byte, error) {
	var buf bytes.Buffer

	// 写入 [common] 节
	if config.Common != nil {
		buf.WriteString("[common]\n")
		if config.Common.ServerAddr != "" {
			buf.WriteString(fmt.Sprintf("server_addr = %s\n", config.Common.ServerAddr))
		}
		if config.Common.ServerPort > 0 {
			buf.WriteString(fmt.Sprintf("server_port = %d\n", config.Common.ServerPort))
		}
		if config.Common.AuthToken != "" {
			buf.WriteString(fmt.Sprintf("auth_token = %s\n", config.Common.AuthToken))
		}
		if config.Common.AuthMethod != "" {
			buf.WriteString(fmt.Sprintf("auth_method = %s\n", config.Common.AuthMethod))
		}
		if config.Common.User != "" {
			buf.WriteString(fmt.Sprintf("user = %s\n", config.Common.User))
		}
		if config.Common.Protocol != "" {
			buf.WriteString(fmt.Sprintf("protocol = %s\n", config.Common.Protocol))
		}
		if config.Common.TLSEnable {
			buf.WriteString("tls_enable = true\n")
		}
		buf.WriteString("\n")
	}

	// 写入代理规则
	for _, proxy := range config.Proxies {
		buf.WriteString(fmt.Sprintf("[%s]\n", proxy.Name))
		if proxy.Type != "" {
			buf.WriteString(fmt.Sprintf("type = %s\n", proxy.Type))
		}
		if proxy.LocalIP != "" {
			buf.WriteString(fmt.Sprintf("local_ip = %s\n", proxy.LocalIP))
		}
		if proxy.LocalPort > 0 {
			buf.WriteString(fmt.Sprintf("local_port = %d\n", proxy.LocalPort))
		}
		if proxy.RemotePort > 0 {
			buf.WriteString(fmt.Sprintf("remote_port = %d\n", proxy.RemotePort))
		}
		if proxy.Subdomain != "" {
			buf.WriteString(fmt.Sprintf("subdomain = %s\n", proxy.Subdomain))
		}
		if proxy.Sk != "" {
			buf.WriteString(fmt.Sprintf("sk = %s\n", proxy.Sk))
		}
		if len(proxy.CustomDomains) > 0 {
			buf.WriteString(fmt.Sprintf("custom_domains = %s\n", strings.Join(proxy.CustomDomains, ",")))
		}
		buf.WriteString("\n")
	}

	return buf.Bytes(), nil
}

// Extension 获取文件扩展名
func (p *INIParser) Extension() string {
	return ".ini"
}
