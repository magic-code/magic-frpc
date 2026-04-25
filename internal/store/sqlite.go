// Package store 提供 SQLite 数据存储
package store

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/magic-frpc/gui/pkg/types"
	_ "modernc.org/sqlite"
)

// SQLiteStore SQLite 存储实现
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore 创建 SQLite 存储
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// 设置连接池参数
	db.SetMaxOpenConns(1) // SQLite 只支持单写连接
	db.SetMaxIdleConns(1)

	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		return nil, err
	}

	return store, nil
}

// migrate 执行数据库迁移
func (s *SQLiteStore) migrate() error {
	// 创建配置表
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS configs (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			tags TEXT DEFAULT '[]',
			auto_start_on_launch BOOLEAN DEFAULT FALSE,
			data TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Printf("创建配置表失败: %v", err)
		return err
	}

	// 添加 auto_start_on_launch 列（迁移已有数据库）
	_, _ = s.db.Exec(`ALTER TABLE configs ADD COLUMN auto_start_on_launch BOOLEAN DEFAULT FALSE`)

	// 创建版本表
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS versions (
			version TEXT PRIMARY KEY,
			path TEXT NOT NULL,
			is_active BOOLEAN DEFAULT FALSE,
			installed_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Printf("创建版本表失败: %v", err)
		return err
	}

	// 创建设置表
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)
	`)
	if err != nil {
		log.Printf("创建设置表失败: %v", err)
		return err
	}

	return nil
}

// Close 关闭数据库连接
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// DB 获取数据库连接
func (s *SQLiteStore) DB() *sql.DB {
	return s.db
}

// ========== 配置存储方法 ==========

// ConfigList 获取配置列表（元数据）
func (s *SQLiteStore) ConfigList() ([]*types.ConfigMeta, error) {
	rows, err := s.db.Query(`
		SELECT id, name, description, tags, auto_start_on_launch, created_at, updated_at
		FROM configs
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*types.ConfigMeta
	for rows.Next() {
		var config types.ConfigMeta
		var tagsJSON string
		var createdAt, updatedAt string
		var autoStartOnLaunch bool

		if err := rows.Scan(&config.ID, &config.Name, &config.Description, &tagsJSON, &autoStartOnLaunch, &createdAt, &updatedAt); err != nil {
			return nil, err
		}

		config.AutoStartOnLaunch = autoStartOnLaunch

		// 解析标签
		if err := json.Unmarshal([]byte(tagsJSON), &config.Tags); err != nil {
			config.Tags = []string{}
		}

		// 解析时间
		if t, err := time.Parse("2006-01-02 15:04:05", createdAt); err == nil {
			config.CreatedAt = t
		}
		if t, err := time.Parse("2006-01-02 15:04:05", updatedAt); err == nil {
			config.UpdatedAt = t
		}

		configs = append(configs, &config)
	}

	return configs, nil
}

// ConfigGet 获取配置详情
func (s *SQLiteStore) ConfigGet(id string) (*types.FrpcConfig, error) {
	var data string
	var tagsJSON string
	var createdAt, updatedAt string
	var autoStartOnLaunch bool

	err := s.db.QueryRow(`
		SELECT data, tags, auto_start_on_launch, created_at, updated_at
		FROM configs
		WHERE id = ?
	`, id).Scan(&data, &tagsJSON, &autoStartOnLaunch, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	log.Printf("读取配置数据: %s", data)

	// 解析 JSON 数据
	var config types.FrpcConfig
	if err := json.Unmarshal([]byte(data), &config); err != nil {
		return nil, err
	}

	log.Printf("解析后代理数量: %d", len(config.Proxies))

	// 确保 ID 一致
	config.ID = id

	// 设置自动启动标记
	config.AutoStartOnLaunch = autoStartOnLaunch

	// 解析标签
	if err := json.Unmarshal([]byte(tagsJSON), &config.Tags); err != nil {
		config.Tags = []string{}
	}

	// 解析时间
	if t, err := time.Parse("2006-01-02 15:04:05", createdAt); err == nil {
		config.CreatedAt = t
	}
	if t, err := time.Parse("2006-01-02 15:04:05", updatedAt); err == nil {
		config.UpdatedAt = t
	}

	return &config, nil
}

// ConfigSave 保存配置
func (s *SQLiteStore) ConfigSave(config *types.FrpcConfig) error {
	// 序列化配置数据（不含元数据字段）
	configData := map[string]interface{}{
		"name":        config.Name,
		"description": config.Description,
		"common":      config.Common,
		"proxies":     config.Proxies,
	}

	data, err := json.Marshal(configData)
	if err != nil {
		return err
	}

	// 序列化标签
	tagsJSON, err := json.Marshal(config.Tags)
	if err != nil {
		return err
	}

	now := time.Now().Format("2006-01-02 15:04:05")

	// 使用 UPSERT 语义
	_, err = s.db.Exec(`
		INSERT INTO configs (id, name, description, tags, auto_start_on_launch, data, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			tags = excluded.tags,
			auto_start_on_launch = excluded.auto_start_on_launch,
			data = excluded.data,
			updated_at = excluded.updated_at
	`, config.ID, config.Name, config.Description, string(tagsJSON), config.AutoStartOnLaunch, string(data), now, now)

	return err
}

// ConfigDelete 删除配置
func (s *SQLiteStore) ConfigDelete(id string) error {
	_, err := s.db.Exec(`DELETE FROM configs WHERE id = ?`, id)
	return err
}

// ========== 设置存储方法 ==========

// GetSetting 获取设置值
func (s *SQLiteStore) GetSetting(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// SetSetting 保存设置值
func (s *SQLiteStore) SetSetting(key, value string) error {
	_, err := s.db.Exec(`
		INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	return err
}

// ========== 版本存储方法 ==========

// VersionList 获取本地版本列表
func (s *SQLiteStore) VersionList() ([]*types.LocalVersion, error) {
	rows, err := s.db.Query(`
		SELECT version, path, is_active, installed_at
		FROM versions
		ORDER BY installed_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []*types.LocalVersion
	for rows.Next() {
		var v types.LocalVersion
		var isActive bool
		var installedAt string

		if err := rows.Scan(&v.Version, &v.Path, &isActive, &installedAt); err != nil {
			return nil, err
		}

		v.IsActive = isActive
		if t, err := time.Parse("2006-01-02 15:04:05", installedAt); err == nil {
			v.InstalledAt = t
		}

		versions = append(versions, &v)
	}

	return versions, nil
}

// VersionSave 保存版本信息
func (s *SQLiteStore) VersionSave(version, path string, isActive bool) error {
	now := time.Now().Format("2006-01-02 15:04:05")

	// 如果设为活跃，先取消其他活跃版本
	if isActive {
		_, _ = s.db.Exec(`UPDATE versions SET is_active = FALSE WHERE is_active = TRUE`)
	}

	_, err := s.db.Exec(`
		INSERT INTO versions (version, path, is_active, installed_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(version) DO UPDATE SET
			path = excluded.path,
			is_active = excluded.is_active
	`, version, path, isActive, now)

	return err
}

// VersionSetActive 设置活跃版本
func (s *SQLiteStore) VersionSetActive(version string) error {
	// 取消所有活跃版本
	if _, err := s.db.Exec(`UPDATE versions SET is_active = FALSE`); err != nil {
		return err
	}

	// 设置指定版本为活跃
	_, err := s.db.Exec(`UPDATE versions SET is_active = TRUE WHERE version = ?`, version)
	return err
}

// VersionGetActive 获取活跃版本
func (s *SQLiteStore) VersionGetActive() (*types.LocalVersion, error) {
	var v types.LocalVersion
	var isActive bool
	var installedAt string

	err := s.db.QueryRow(`
		SELECT version, path, is_active, installed_at
		FROM versions
		WHERE is_active = TRUE
	`).Scan(&v.Version, &v.Path, &isActive, &installedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	v.IsActive = isActive
	if t, err := time.Parse("2006-01-02 15:04:05", installedAt); err == nil {
		v.InstalledAt = t
	}

	return &v, nil
}

// VersionDelete 删除版本记录
func (s *SQLiteStore) VersionDelete(version string) error {
	_, err := s.db.Exec(`DELETE FROM versions WHERE version = ?`, version)
	return err
}
