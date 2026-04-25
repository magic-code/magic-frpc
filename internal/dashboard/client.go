// Package dashboard 提供 frpc Dashboard API 客户端
package dashboard

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/magic-frpc/gui/pkg/types"
)

// Client frpc Dashboard API 客户端
type Client struct {
	baseURL    string
	httpClient *http.Client
	username   string
	password   string
}

// NewClient 创建 Dashboard API 客户端
func NewClient(host string, port int, username, password string) *Client {
	// 如果没有提供用户名密码，使用默认值
	if username == "" {
		username = "admin"
	}
	if password == "" {
		password = "admin"
	}
	return &Client{
		baseURL: fmt.Sprintf("http://%s:%d", host, port),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		username: username,
		password: password,
	}
}

// doRequest 发送带认证的请求
func (c *Client) doRequest(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	// 设置 Basic Auth
	req.SetBasicAuth(c.username, c.password)
	return c.httpClient.Do(req)
}

// trafficResponse /api/traffic/{name} 返回的流量信息
// frpc Dashboard API 响应格式：
//
//	{
//	  "name": "proxy_name",
//	  "trafficIn": [30732,0,0,0,0,0,0],  // 数组，第一个元素是今日流量
//	  "trafficOut": [8254870,0,0,0,0,0,0]
//	}
type trafficResponse struct {
	Name       string  `json:"name"`
	TrafficIn  []int64 `json:"trafficIn"`  // 流量入数组，第一个元素为今日流量
	TrafficOut []int64 `json:"trafficOut"` // 流量出数组，第一个元素为今日流量
}

// statusResponse /api/status 返回的状态信息
type statusResponse struct {
	Version    string `json:"version"`
	Opetime    string `json:"opetime"` // 运行时间
	CpuUsage   string `json:"cpu_usage"`
	MemUsage   string `json:"mem_usage"`
	ClientNum  int    `json:"client_num"`
	Connection int    `json:"connection"` // 连接数
}

// GetTraffic 获取指定代理的流量信息
func (c *Client) GetTraffic(proxyName string) (*types.ProxyStatus, error) {
	fullURL := fmt.Sprintf("%s/api/traffic/%s", c.baseURL, proxyName)
	log.Printf("[Dashboard] 请求流量接口: %s", fullURL)
	resp, err := c.doRequest(fullURL)
	if err != nil {
		return nil, fmt.Errorf("请求 %s 失败: %w", fullURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s 返回状态码: %d", fullURL, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	log.Printf("[Dashboard] %s 响应: %s", fullURL, truncate(string(body), 500))

	var tr trafficResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 从数组中提取今日流量（第一个元素）
	var todayTrafficIn, todayTrafficOut int64
	if len(tr.TrafficIn) > 0 {
		todayTrafficIn = tr.TrafficIn[0]
	}
	if len(tr.TrafficOut) > 0 {
		todayTrafficOut = tr.TrafficOut[0]
	}

	return &types.ProxyStatus{
		Name:            tr.Name,
		TodayTrafficIn:  todayTrafficIn,
		TodayTrafficOut: todayTrafficOut,
		Status:          "running", // 有流量数据说明在运行
	}, nil
}

// GetAllTraffic 获取所有代理的流量信息
// 需要知道代理名称列表，然后逐个查询
func (c *Client) GetAllTraffic(proxyNames []string) ([]types.ProxyStatus, int64, int64, error) {
	var result []types.ProxyStatus
	var totalIn, totalOut int64

	for _, name := range proxyNames {
		status, err := c.GetTraffic(name)
		if err != nil {
			log.Printf("[Dashboard] 获取代理 %s 流量失败: %v", name, err)
			// 失败时添加一个空状态
			result = append(result, types.ProxyStatus{
				Name:   name,
				Status: "offline",
			})
			continue
		}

		result = append(result, *status)
		totalIn += status.TodayTrafficIn
		totalOut += status.TodayTrafficOut
	}

	return result, totalIn, totalOut, nil
}

// GetProxyStatuses 获取代理状态（合并流量信息）- 兼容旧接口
func (c *Client) GetProxyStatuses() ([]types.ProxyStatus, int64, int64, error) {
	// 这个方法需要代理名称列表，但当前不知道
	// 返回空结果，让调用方提供代理名称
	log.Printf("[Dashboard] GetProxyStatuses 被调用，但没有代理名称列表")
	return nil, 0, 0, fmt.Errorf("需要提供代理名称列表")
}

// GetProxyStatusesWithNames 获取指定代理列表的流量状态
func (c *Client) GetProxyStatusesWithNames(proxyNames []string) ([]types.ProxyStatus, int64, int64, error) {
	return c.GetAllTraffic(proxyNames)
}

// CheckHealth 检查 Dashboard 是否可用
func (c *Client) CheckHealth() bool {
	// frpc 客户端使用 /api/status 端点
	url := fmt.Sprintf("%s/api/status", c.baseURL)
	resp, err := c.doRequest(url)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// truncate 截断字符串用于日志输出
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
