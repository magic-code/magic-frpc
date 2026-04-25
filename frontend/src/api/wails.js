/**
 * Wails v3 API 封装
 * 使用自动生成的绑定调用后端服务
 */

// 导入 Wails v3 自动生成的绑定
import * as App from '../../bindings/github.com/magic-frpc/gui/internal/app/app.js'

// ========== 配置 API ==========
export const configApi = {
  list: () => App.ConfigList(),
  get: (id) => App.ConfigGet(id),
  save: (config) => App.ConfigSave(JSON.stringify(config)),
  delete: (id) => App.ConfigDelete(id),
  validate: (config) => App.ConfigValidate(JSON.stringify(config)),
  new: (name) => App.ConfigNew(name),
  import: (name, data, format) => App.ConfigImport(name, data, format),
  export: (id, format) => App.ConfigExport(id, format),
  serialize: (config, format) => App.ConfigSerialize(JSON.stringify(config), format),
  parse: (source, format) => App.ConfigParse(source, format),
}

// ========== frpc 进程 API ==========
export const frpcApi = {
  start: (configId) => App.FrpcStart(configId),
  stop: (configId) => App.FrpcStop(configId),
  restart: (configId) => App.FrpcRestart(configId),
  getStatus: (configId) => App.FrpcGetStatus(configId),
  getAllStatus: () => App.FrpcGetAllStatus(),
  getLogs: (configId, limit) => App.FrpcGetLogs(configId, limit),
  clearLogs: (configId) => App.FrpcClearLogs(configId),
}

// ========== 版本管理 API ==========
export const versionApi = {
  listRemote: () => App.VersionListRemote(),
  listLocal: () => App.VersionListLocal(),
  download: (version) => App.VersionDownload(version),
  setActive: (version) => App.VersionSetActive(version),
  getActive: () => App.VersionGetActive(),
  delete: (version) => App.VersionDelete(version),
  getActiveFrpcPath: () => App.VersionGetActiveFrpcPath(),
}

// ========== 平台 API ==========
export const platformApi = {
  getInfo: () => App.PlatformGetInfo(),
}

// ========== 设置 API ==========
export const settingsApi = {
  get: () => App.GetSettings(),
  save: (settings) => App.SaveSettings(JSON.stringify(settings)),
}

// ========== 应用日志 API ==========
export const appApi = {
  getLogs: (limit) => App.AppGetLogs(limit),
  clearLogs: () => App.AppClearLogs(),
  getDataDir: () => App.GetDataDir(),
}
