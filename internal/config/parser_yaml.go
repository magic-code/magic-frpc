// Package config 提供配置解析器
package config

import (
	"fmt"

	"github.com/magic-frpc/gui/pkg/types"
	"gopkg.in/yaml.v3"
)

// YAMLParser YAML 配置解析器
type YAMLParser struct{}

// NewYAMLParser 创建 YAML 解析器
func NewYAMLParser() *YAMLParser {
	return &YAMLParser{}
}

// Parse 解析 YAML 配置
func (p *YAMLParser) Parse(data []byte) (*types.FrpcConfig, error) {
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("解析 YAML 失败: %w", err)
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
		if v, ok := common["server_addr"].(string); ok {
			config.Common.ServerAddr = v
		}
		if v, ok := common["server_port"].(int); ok {
			config.Common.ServerPort = v
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
	}

	// 解析代理规则
	if proxies, ok := raw["proxies"].([]interface{}); ok {
		for _, item := range proxies {
			if proxyMap, ok := item.(map[string]interface{}); ok {
				proxy := types.ProxyConfig{}
				if v, ok := proxyMap["name"].(string); ok {
					proxy.Name = v
				}
				if v, ok := proxyMap["type"].(string); ok {
					proxy.Type = v
				}
				if v, ok := proxyMap["local_ip"].(string); ok {
					proxy.LocalIP = v
				}
				if v, ok := proxyMap["local_port"].(int); ok {
					proxy.LocalPort = v
				}
				if v, ok := proxyMap["remote_port"].(int); ok {
					proxy.RemotePort = v
				}
				if v, ok := proxyMap["subdomain"].(string); ok {
					proxy.Subdomain = v
				}
				if v, ok := proxyMap["sk"].(string); ok {
					proxy.Sk = v
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
	}

	return config, nil
}

// Serialize 序列化为 YAML 格式
func (p *YAMLParser) Serialize(config *types.FrpcConfig) ([]byte, error) {
	output := make(map[string]interface{})

	// 写入 common 节
	if config.Common != nil {
		output["common"] = map[string]interface{}{
			"server_addr": config.Common.ServerAddr,
			"server_port": config.Common.ServerPort,
			"auth_token":  config.Common.AuthToken,
			"auth_method": config.Common.AuthMethod,
			"user":        config.Common.User,
			"protocol":    config.Common.Protocol,
			"tls_enable":  config.Common.TLSEnable,
		}
	}

	// 写入代理规则
	if len(config.Proxies) > 0 {
		var proxies []map[string]interface{}
		for _, proxy := range config.Proxies {
			p := map[string]interface{}{
				"name":       proxy.Name,
				"type":       proxy.Type,
				"local_ip":   proxy.LocalIP,
				"local_port": proxy.LocalPort,
			}
			if proxy.RemotePort > 0 {
				p["remote_port"] = proxy.RemotePort
			}
			if proxy.Subdomain != "" {
				p["subdomain"] = proxy.Subdomain
			}
			if proxy.Sk != "" {
				p["sk"] = proxy.Sk
			}
			if len(proxy.CustomDomains) > 0 {
				p["custom_domains"] = proxy.CustomDomains
			}
			proxies = append(proxies, p)
		}
		output["proxies"] = proxies
	}

	data, err := yaml.Marshal(output)
	if err != nil {
		return nil, fmt.Errorf("序列化 YAML 失败: %w", err)
	}

	return data, nil
}

// Extension 获取文件扩展名
func (p *YAMLParser) Extension() string {
	return ".yaml"
}
