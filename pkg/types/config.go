// Package types 定义核心数据模型
package types

import "time"

// FrpcConfig frpc 配置文件结构
type FrpcConfig struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	Description       string         `json:"description"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
	Tags              []string       `json:"tags"`
	Common            *CommonSection `json:"common"`
	Proxies           []ProxyConfig  `json:"proxies"`
	AutoStartOnLaunch bool           `json:"autoStartOnLaunch"` // 程序启动时自动启动此配置
}

// ConfigMeta 配置文件元数据
type ConfigMeta struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
	Tags              []string  `json:"tags"`
	AutoStartOnLaunch bool      `json:"autoStartOnLaunch"` // 程序启动时自动启动此配置
}

// CommonSection frpc [common] 配置节
// 参考: https://gofrp.org/zh-cn/docs/client-config/
type CommonSection struct {
	// 服务器配置
	ServerAddr string `json:"serverAddr"`
	ServerPort int    `json:"serverPort"`

	// 认证配置
	AuthToken  string `json:"authToken"`
	AuthMethod string `json:"authMethod"` // token, oidc
	User       string `json:"user"`       // 用户名，用于区分不同客户端

	// 传输配置
	Protocol   string `json:"protocol"`   // tcp, kcp, websocket, wss, quic
	TLSEnable  bool   `json:"tlsEnable"`  // 启用 TLS
	ServerName string `json:"serverName"` // TLS Server Name，用于证书验证

	// 心跳配置
	HeartbeatInterval int `json:"heartbeatInterval"` // 心跳间隔（秒）
	HeartbeatTimeout  int `json:"heartbeatTimeout"`  // 心跳超时（秒）

	// Dashboard 配置（用于获取代理状态和流量）
	AdminAddr     string `json:"adminAddr"`     // Dashboard 地址，默认使用服务器地址
	AdminPort     int    `json:"adminPort"`     // Dashboard 端口，默认 7500
	AdminUser     string `json:"adminUser"`     // Dashboard 用户名（可选）
	AdminPassword string `json:"adminPassword"` // Dashboard 密码（可选）

	// HTTP 代理访问配置（可选）
	SubDomainHost string `json:"subDomainHost"` // 子域名后缀，如 frp.example.com，用于构建完整访问链接
	HttpPort      int    `json:"httpPort"`      // HTTP 访问端口，默认 80
	HttpsPort     int    `json:"httpsPort"`     // HTTPS 访问端口，默认 443

	// 其他配置
	Transport struct {
		ProxyURL string `json:"proxyUrl"` // 代理 URL
	} `json:"transport"`
}

// ProxyConfig 代理配置
// 参考: https://gofrp.org/zh-cn/docs/proxy/
type ProxyConfig struct {
	Name              string            `json:"name"`
	Type              string            `json:"type"`              // tcp, udp, http, https, stcp, sudp, xtcp, xudp, tcpmux
	LocalIP           string            `json:"localIP"`           // 本地 IP
	LocalPort         int               `json:"localPort"`         // 本地端口
	RemotePort        int               `json:"remotePort"`        // 远程端口（TCP/UDP 类型需要）
	CustomDomains     []string          `json:"customDomains"`     // 自定义域名（HTTP/HTTPS 类型）
	Subdomain         string            `json:"subdomain"`         // 子域名（HTTP/HTTPS 类型）
	Sk                string            `json:"sk"`                // 访问密钥（STCP/SUDP 类型）
	HostHeaderRewrite string            `json:"hostHeaderRewrite"` // 重写 Host 头（HTTP 类型）
	Headers           map[string]string `json:"headers"`           // 自定义请求头（HTTP 类型）
}
