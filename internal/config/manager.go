// Package config 提供配置管理功能
package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/magic-frpc/gui/internal/crypto"
	"github.com/magic-frpc/gui/pkg/types"
)

// Parser 配置解析器接口
type Parser interface {
	Parse(data []byte) (*types.FrpcConfig, error)
	Serialize(config *types.FrpcConfig) ([]byte, error)
	Extension() string
}

// Store 配置存储接口
type Store interface {
	ConfigList() ([]*types.ConfigMeta, error)
	ConfigGet(id string) (*types.FrpcConfig, error)
	ConfigSave(config *types.FrpcConfig) error
	ConfigDelete(id string) error
}

// Manager 配置管理器
type Manager struct {
	store     Store
	validator *Validator
	encryptor crypto.Encryptor
	parsers   map[string]Parser
}

// NewManager 创建配置管理器
func NewManager(store Store) *Manager {
	m := &Manager{
		store:     store,
		validator: NewValidator(),
		encryptor: crypto.NewAESEncryptor(),
		parsers:   make(map[string]Parser),
	}

	// 注册解析器
	m.parsers["toml"] = NewTOMLParser()
	m.parsers["ini"] = NewINIParser()
	m.parsers["yaml"] = NewYAMLParser()

	return m
}

// List 获取配置列表
func (m *Manager) List() ([]*types.ConfigMeta, error) {
	return m.store.ConfigList()
}

// Get 获取配置详情
func (m *Manager) Get(id string) (*types.FrpcConfig, error) {
	config, err := m.store.ConfigGet(id)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, fmt.Errorf("配置不存在: %s", id)
	}

	// 解密敏感字段
	if config.Common != nil && config.Common.AuthToken != "" {
		decrypted, err := m.encryptor.Decrypt(config.Common.AuthToken)
		if err == nil {
			config.Common.AuthToken = decrypted
		}
	}

	return config, nil
}

// Save 保存配置
func (m *Manager) Save(config *types.FrpcConfig) (*ValidationResult, error) {
	// 验证配置
	result := m.validator.Validate(config)
	if !result.Valid {
		return result, fmt.Errorf("配置验证失败: %s", strings.Join(result.Errors, "; "))
	}

	// 设置 ID 和时间戳
	if config.ID == "" {
		config.ID = uuid.New().String()
	}
	now := time.Now()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = now
	}
	config.UpdatedAt = now

	// 加密敏感字段（创建副本）
	encryptedConfig := *config
	if config.Common != nil {
		encryptedCommon := *config.Common
		if encryptedCommon.AuthToken != "" {
			encrypted, err := m.encryptor.Encrypt(encryptedCommon.AuthToken)
			if err != nil {
				return result, fmt.Errorf("加密认证令牌失败: %w", err)
			}
			encryptedCommon.AuthToken = encrypted
		}
		encryptedConfig.Common = &encryptedCommon
	}

	// 保存到存储
	if err := m.store.ConfigSave(&encryptedConfig); err != nil {
		return result, fmt.Errorf("保存配置失败: %w", err)
	}

	return result, nil
}

// Delete 删除配置
func (m *Manager) Delete(id string) error {
	return m.store.ConfigDelete(id)
}

// Validate 验证配置（不保存）
func (m *Manager) Validate(config *types.FrpcConfig) *ValidationResult {
	return m.validator.Validate(config)
}

// Import 导入配置文件
func (m *Manager) Import(name string, data []byte, format string) (*types.FrpcConfig, error) {
	// 获取解析器
	parser, ok := m.parsers[strings.ToLower(format)]
	if !ok {
		return nil, fmt.Errorf("不支持的配置格式: %s", format)
	}

	// 解析配置
	config, err := parser.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// 设置名称
	config.Name = name

	// 验证配置
	result := m.validator.Validate(config)
	if !result.Valid {
		return nil, fmt.Errorf("配置验证失败: %s", strings.Join(result.Errors, "; "))
	}

	return config, nil
}

// Export 导出配置文件
func (m *Manager) Export(id string, format string) ([]byte, string, error) {
	// 获取配置
	config, err := m.Get(id)
	if err != nil {
		return nil, "", err
	}

	// 获取解析器
	parser, ok := m.parsers[strings.ToLower(format)]
	if !ok {
		return nil, "", fmt.Errorf("不支持的配置格式: %s", format)
	}

	// 序列化配置
	data, err := parser.Serialize(config)
	if err != nil {
		return nil, "", fmt.Errorf("序列化配置失败: %w", err)
	}

	// 生成文件名
	filename := fmt.Sprintf("%s%s", config.Name, parser.Extension())

	return data, filename, nil
}

// DetectFormat 检测配置文件格式
func (m *Manager) DetectFormat(filename string, data []byte) string {
	// 根据文件扩展名判断
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".toml":
		return "toml"
	case ".ini", ".conf":
		return "ini"
	case ".yaml", ".yml":
		return "yaml"
	}

	// 根据内容判断
	content := strings.TrimSpace(string(data))
	if strings.HasPrefix(content, "[") && strings.Contains(content, "[common]") {
		return "ini"
	}
	if strings.Contains(content, ":") && !strings.HasPrefix(content, "[") {
		return "yaml"
	}

	// 默认使用 TOML
	return "toml"
}

// NewConfig 创建新的空配置
func (m *Manager) NewConfig(name string) *types.FrpcConfig {
	return &types.FrpcConfig{
		ID:          uuid.New().String(),
		Name:        name,
		Description: "",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Tags:        []string{},
		Common: &types.CommonSection{
			ServerAddr: "",
			ServerPort: 7000,
			Protocol:   "tcp",
			TLSEnable:  false,
		},
		Proxies: []types.ProxyConfig{},
	}
}

// SerializeConfig 序列化配置到指定格式
func (m *Manager) SerializeConfig(config *types.FrpcConfig, format string) ([]byte, error) {
	parser, ok := m.parsers[strings.ToLower(format)]
	if !ok {
		return nil, fmt.Errorf("不支持的配置格式: %s", format)
	}
	return parser.Serialize(config)
}

// ParseConfig 从指定格式解析配置
func (m *Manager) ParseConfig(data []byte, format string) (*types.FrpcConfig, error) {
	parser, ok := m.parsers[strings.ToLower(format)]
	if !ok {
		return nil, fmt.Errorf("不支持的配置格式: %s", format)
	}
	return parser.Parse(data)
}

// ToJSON 将配置转换为 JSON 字符串
func (m *Manager) ToJSON(config *types.FrpcConfig) (string, error) {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON 从 JSON 字符串解析配置
func (m *Manager) FromJSON(jsonStr string) (*types.FrpcConfig, error) {
	var config types.FrpcConfig
	if err := json.Unmarshal([]byte(jsonStr), &config); err != nil {
		return nil, err
	}
	return &config, nil
}
