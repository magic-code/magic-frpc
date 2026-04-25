<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useConfigStore } from '@/stores/config'
import { useProcessStore } from '@/stores/process'
import { useVersionStore } from '@/stores/version'

const { t } = useI18n()
const configStore = useConfigStore()
const processStore = useProcessStore()
const versionStore = useVersionStore()
const selectedConfigId = ref(null)
const editingConfig = ref(null)
const showDeleteDialog = ref(false)
const deleteTargetId = ref(null)
const deleteTargetName = ref('')
const showImportDialog = ref(false)
const importName = ref('')
const importContent = ref('')
const importFormat = ref('toml')
const saveMessage = ref('')
const isSaving = ref(false)

// 表单数据
const formData = ref({
  name: '',
  description: '',
  tags: [],
  common: null,
  proxies: [],
  autoStartOnLaunch: false,
})

// 编辑代理索引（-1 表示新增，>=0 表示编辑）
const editingProxyIndex = ref(-1)

// 新代理表单
const newProxy = ref({
  name: '',
  type: 'tcp',
  localIP: '127.0.0.1',
  localPort: 80,
  remotePort: 0,
  subdomain: '',
})

onMounted(async () => {
  await configStore.load()
  await processStore.refreshStatus()
  await versionStore.init()
})

const selectedConfig = computed(() => {
  if (!selectedConfigId.value) return null
  return configStore.configs.find(c => c.id === selectedConfigId.value)
})

const processStatus = computed(() => {
  if (!selectedConfigId.value) return null
  return processStore.getStatus(selectedConfigId.value)
})

function handleSelect(id) {
  selectedConfigId.value = id
  loadConfigForEdit(id)
}

async function loadConfigForEdit(id) {
  try {
    const config = await configStore.get(id)
    if (config) {
      editingConfig.value = config
      formData.value = {
        name: config.name || '',
        description: config.description || '',
        tags: config.tags || [],
        common: config.common ? {
          serverAddr: config.common.serverAddr || '',
          serverPort: config.common.serverPort || 7000,
          authToken: config.common.authToken || '',
          authMethod: config.common.authMethod || 'token',
          user: config.common.user || '',
          protocol: config.common.protocol || 'tcp',
          tlsEnable: config.common.tlsEnable || false,
          serverName: config.common.serverName || '',
          heartbeatInterval: config.common.heartbeatInterval || 10,
          heartbeatTimeout: config.common.heartbeatTimeout || 90,
          subDomainHost: config.common.subDomainHost || '',
          httpPort: config.common.httpPort || 80,
          httpsPort: config.common.httpsPort || 443,
          adminPort: config.common.adminPort || 7500,
          adminUser: config.common.adminUser || '',
          adminPassword: config.common.adminPassword || '',
        } : null,
        proxies: config.proxies || [],
        autoStartOnLaunch: config.autoStartOnLaunch || false,
      }
    }
  } catch (err) {
    console.error('Failed to load config:', err)
  }
}

async function handleCreate() {
  try {
    const config = await configStore.create(t('config.newConfig'))
    if (config && config.id) {
      await configStore.load()
      selectedConfigId.value = config.id
      loadConfigForEdit(config.id)
    }
  } catch (err) {
    alert(t('common.error') + ': ' + err.message)
  }
}

function handleDeleteRequest(id) {
  const config = configStore.configs.find(c => c.id === id)
  deleteTargetId.value = id
  deleteTargetName.value = config?.name || ''
  showDeleteDialog.value = true
}

async function handleDeleteConfirm() {
  if (deleteTargetId.value) {
    await configStore.deleteConfig(deleteTargetId.value)
    if (selectedConfigId.value === deleteTargetId.value) {
      selectedConfigId.value = null
      editingConfig.value = null
    }
    showDeleteDialog.value = false
    deleteTargetId.value = null
  }
}

async function handleSave() {
  if (!editingConfig.value) return
  isSaving.value = true
  saveMessage.value = ''

  try {
    const configToSave = {
      ...editingConfig.value,
      name: formData.value.name,
      description: formData.value.description,
      tags: formData.value.tags,
      common: formData.value.common,
      proxies: formData.value.proxies,
      autoStartOnLaunch: formData.value.autoStartOnLaunch,
      updatedAt: new Date().toISOString(),
    }

    await configStore.save(configToSave)
    await configStore.load()
    saveMessage.value = t('common.success')
    setTimeout(() => { saveMessage.value = '' }, 2000)
  } catch (err) {
    alert(t('common.error') + ': ' + err.message)
    console.error('Save failed:', err)
  } finally {
    isSaving.value = false
  }
}

function initCommonConfig() {
  if (!formData.value.common) {
    formData.value.common = {
      serverAddr: '',
      serverPort: 7000,
      authToken: '',
      authMethod: 'token',
      user: '',
      protocol: 'tcp',
      tlsEnable: false,
      serverName: '',
      heartbeatInterval: 10,
      heartbeatTimeout: 90,
      subDomainHost: '',
      httpPort: 80,
      httpsPort: 443,
      adminPort: 7500,
      adminUser: '',
      adminPassword: '',
    }
  }
}

function handleAddProxy() {
  if (!newProxy.value.name) {
    alert(t('config.proxyName'))
    return
  }
  if (!newProxy.value.localPort) {
    alert(t('config.localPort'))
    return
  }

  initCommonConfig()

  if (editingProxyIndex.value >= 0) {
    // 编辑模式：更新现有代理
    formData.value.proxies[editingProxyIndex.value] = { ...newProxy.value }
    editingProxyIndex.value = -1
  } else {
    // 新增模式
    formData.value.proxies.push({ ...newProxy.value })
  }

  // 重置表单
  newProxy.value = {
    name: '',
    type: 'tcp',
    localIP: '127.0.0.1',
    localPort: 80,
    remotePort: 0,
    subdomain: '',
  }
}

function handleEditProxy(index) {
  const proxy = formData.value.proxies[index]
  newProxy.value = { ...proxy }
  editingProxyIndex.value = index
}

function handleCancelEditProxy() {
  editingProxyIndex.value = -1
  newProxy.value = {
    name: '',
    type: 'tcp',
    localIP: '127.0.0.1',
    localPort: 80,
    remotePort: 0,
    subdomain: '',
  }
}

function handleRemoveProxy(index) {
  formData.value.proxies.splice(index, 1)
  // 如果正在编辑被删除的代理，取消编辑
  if (editingProxyIndex.value === index) {
    editingProxyIndex.value = -1
    newProxy.value = {
      name: '',
      type: 'tcp',
      localIP: '127.0.0.1',
      localPort: 80,
      remotePort: 0,
      subdomain: '',
    }
  }
}

async function handleStart() {
  if (!selectedConfigId.value) return
  if (!versionStore.hasActiveVersion) {
    alert(t('status.pleaseSetVersion'))
    return
  }
  try {
    await processStore.start(selectedConfigId.value)
  } catch (err) {
    alert(t('status.start') + ' ' + t('common.error').toLowerCase() + ': ' + err.message)
  }
}

async function handleStop() {
  if (!selectedConfigId.value) return
  try {
    await processStore.stop(selectedConfigId.value)
  } catch (err) {
    alert(t('status.stop') + ' ' + t('common.error').toLowerCase() + ': ' + err.message)
  }
}

async function handleRestart() {
  if (!selectedConfigId.value) return
  try {
    await processStore.restart(selectedConfigId.value)
  } catch (err) {
    alert(t('config.restart') + ' ' + t('common.error').toLowerCase() + ': ' + err.message)
  }
}

function openImportDialog() {
  importName.value = ''
  importContent.value = ''
  importFormat.value = 'toml'
  showImportDialog.value = true
}

async function handleImport() {
  if (!importName.value.trim()) {
    alert(t('config.configName'))
    return
  }
  if (!importContent.value.trim()) {
    alert(t('config.configContent'))
    return
  }
  try {
    const config = await configStore.importConfig(importName.value.trim(), importContent.value.trim(), importFormat.value)
    showImportDialog.value = false
    if (config && config.id) {
      await configStore.load()
      selectedConfigId.value = config.id
      loadConfigForEdit(config.id)
    }
  } catch (err) {
    alert(t('config.importConfig') + ' ' + t('common.error').toLowerCase() + ': ' + err.message)
  }
}
</script>

<template>
  <div class="flex h-full">
    <!-- 左侧：配置列表 -->
    <div class="w-72 border-r border-base-300 bg-base-200 flex-shrink-0 flex flex-col">
      <!-- 工具栏 -->
      <div class="flex items-center justify-between p-3 border-b border-base-300">
        <h2 class="font-semibold">{{ t('config.configList') }}</h2>
        <div class="flex gap-1">
          <button class="btn btn-ghost btn-sm" @click="openImportDialog" :title="t('config.importConfig')">
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
            </svg>
          </button>
          <button class="btn btn-primary btn-sm" @click="handleCreate" :title="t('config.newConfig')">
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
            </svg>
            {{ t('config.create') }}
          </button>
        </div>
      </div>

      <!-- 配置列表 -->
      <div class="flex-1 overflow-auto p-2">
        <div v-if="configStore.isLoading" class="flex items-center justify-center h-32">
          <span class="loading loading-spinner loading-md"></span>
        </div>
        <div v-else-if="configStore.configs.length === 0" class="text-center py-8 text-base-content/50">
          <svg class="h-12 w-12 mx-auto mb-2 opacity-50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          <p>{{ t('config.noConfigs') }}</p>
          <p class="text-sm mt-1">{{ t('config.clickToCreate') }}</p>
        </div>
        <div v-else class="space-y-1">
          <div
            v-for="config in configStore.configs"
            :key="config.id"
            class="group flex items-center justify-between rounded-lg px-3 py-2 cursor-pointer transition-colors"
            :class="selectedConfigId === config.id ? 'bg-primary text-primary-content' : 'hover:bg-base-300'"
            @click="handleSelect(config.id)"
          >
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2">
                <span class="font-medium truncate">{{ config.name }}</span>
                <span
                  v-if="processStore.getStatus(config.id)?.status === 'running'"
                  class="badge badge-success badge-xs"
                >{{ t('config.running') }}</span>
              </div>
              <div class="text-xs opacity-70 truncate">{{ config.description || t('config.noDesc') }}</div>
            </div>
            <button
              class="btn btn-ghost btn-xs opacity-0 group-hover:opacity-100 transition-opacity"
              :class="selectedConfigId === config.id ? 'text-primary-content hover:bg-primary-content/20' : 'text-error'"
              @click.stop="handleDeleteRequest(config.id)"
            >
              <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 右侧：编辑区域 -->
    <div class="flex-1 flex flex-col min-w-0 overflow-hidden">
      <div v-if="selectedConfigId && editingConfig" class="flex-1 overflow-auto p-6">
        <!-- 顶部操作栏 -->
        <div class="flex items-center justify-between mb-6">
          <div class="flex items-center gap-3">
            <h2 class="text-xl font-bold">{{ t('config.configDetail') }}</h2>
            <span v-if="saveMessage" class="badge badge-success">{{ saveMessage }}</span>
          </div>
          <div class="flex gap-2">
            <button
              v-if="processStatus?.status === 'running'"
              class="btn btn-error btn-sm"
              @click="handleStop"
            >
              <svg class="h-4 w-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 10a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z" />
              </svg>
              {{ t('status.stop') }}
            </button>
            <button
              v-else
              class="btn btn-success btn-sm"
              @click="handleStart"
              :disabled="!versionStore.hasActiveVersion"
            >
              <svg class="h-4 w-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              {{ t('status.start') }}
            </button>
            <button class="btn btn-ghost btn-sm" @click="handleRestart" :disabled="processStatus?.status !== 'running'">
              {{ t('config.restart') }}
            </button>
            <button class="btn btn-primary btn-sm" @click="handleSave" :disabled="isSaving">
              <span v-if="isSaving" class="loading loading-spinner loading-xs"></span>
              {{ t('common.save') }}
            </button>
          </div>
        </div>

        <!-- 基本信息 -->
        <div class="card bg-base-200 mb-6">
          <div class="card-body">
            <h3 class="font-medium mb-4">{{ t('config.basicInfo') }}</h3>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-4 max-w-3xl">
              <div class="form-control">
                <label class="label">
                  <span class="label-text font-medium">{{ t('config.configNameRequired') }} <span class="text-error">*</span></span>
                </label>
                <input
                  v-model="formData.name"
                  type="text"
                  class="input input-bordered"
                  :placeholder="t('config.enterConfigName')"
                />
              </div>

              <div class="form-control">
                <label class="label">
                  <span class="label-text font-medium">{{ t('config.desc') }}</span>
                </label>
                <input
                  v-model="formData.description"
                  type="text"
                  class="input input-bordered"
                  :placeholder="t('config.descOptional')"
                />
              </div>
            </div>

            <!-- 自动启动设置 -->
            <div class="form-control mt-4">
              <label class="label cursor-pointer justify-start gap-3">
                <input v-model="formData.autoStartOnLaunch" type="checkbox" class="toggle toggle-primary toggle-sm" />
                <div>
                  <span class="label-text font-medium">{{ t('config.autoStartOnLaunch') }}</span>
                  <p class="text-xs text-base-content/60">{{ t('config.autoStartOnLaunchDesc') }}</p>
                </div>
              </label>
            </div>
          </div>
        </div>

        <!-- 服务器配置 -->
        <div class="divider">{{ t('config.serverConfig') }}</div>
        <div v-if="!formData.common" class="mb-4">
          <button class="btn btn-outline btn-sm" @click="initCommonConfig">
            <svg class="h-4 w-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
            </svg>
            {{ t('config.addServerConfig') }}
          </button>
        </div>
        <div v-else class="grid grid-cols-2 gap-4 max-w-3xl mb-4">
          <div class="form-control">
            <label class="label">
              <span class="label-text">{{ t('config.serverAddrRequired') }} <span class="text-error">*</span></span>
            </label>
            <input
              v-model="formData.common.serverAddr"
              type="text"
              class="input input-bordered"
              placeholder="frp.example.com"
            />
          </div>
          <div class="form-control">
            <label class="label">
              <span class="label-text">{{ t('config.serverPort') }}</span>
            </label>
            <input
              v-model.number="formData.common.serverPort"
              type="number"
              class="input input-bordered"
              placeholder="7000"
            />
          </div>
          <div class="form-control">
            <label class="label">
              <span class="label-text">{{ t('config.authToken') }}</span>
            </label>
            <input
              v-model="formData.common.authToken"
              type="password"
              class="input input-bordered"
              placeholder="your-token"
            />
          </div>
          <div class="form-control">
            <label class="label">
              <span class="label-text">{{ t('config.protocol') }}</span>
            </label>
            <select v-model="formData.common.protocol" class="select select-bordered">
              <option value="tcp">TCP</option>
              <option value="kcp">KCP</option>
              <option value="websocket">WebSocket</option>
              <option value="wss">WSS</option>
              <option value="quic">QUIC</option>
            </select>
          </div>
          <div class="form-control col-span-2">
            <label class="label cursor-pointer justify-start gap-3">
              <input v-model="formData.common.tlsEnable" type="checkbox" class="toggle toggle-primary toggle-sm" />
              <span class="label-text">{{ t('config.enableTLS') }}</span>
            </label>
          </div>

          <!-- HTTP 访问配置（可选，用于生成完整访问链接） -->
          <div class="col-span-2 border-t border-base-300 pt-3 mt-2">
            <h4 class="text-sm font-medium text-base-content/70 mb-2">{{ t('config.httpAccessConfig') }}</h4>
          </div>
          <div class="form-control">
            <label class="label">
              <span class="label-text">{{ t('config.subDomainHost') }}</span>
            </label>
            <input
              v-model="formData.common.subDomainHost"
              type="text"
              class="input input-bordered"
              placeholder="frp.example.com"
            />
          </div>
          <div class="form-control">
            <label class="label">
              <span class="label-text">{{ t('config.httpPort') }}</span>
            </label>
            <input
              v-model.number="formData.common.httpPort"
              type="number"
              class="input input-bordered"
              placeholder="80"
            />
          </div>
          <div class="form-control">
            <label class="label">
              <span class="label-text">{{ t('config.httpsPort') }}</span>
            </label>
            <input
              v-model.number="formData.common.httpsPort"
              type="number"
              class="input input-bordered"
              placeholder="443"
            />
          </div>

          <!-- Dashboard 配置（用于流量监控） -->
          <div class="col-span-2 border-t border-base-300 pt-3 mt-2">
            <h4 class="text-sm font-medium text-base-content/70 mb-2">{{ t('config.dashboardConfig') }}</h4>
          </div>
          <div class="form-control">
            <label class="label">
              <span class="label-text">{{ t('config.dashboardPort') }}</span>
            </label>
            <input
              v-model.number="formData.common.adminPort"
              type="number"
              class="input input-bordered"
              placeholder="7500"
            />
          </div>
          <div class="form-control">
            <label class="label">
              <span class="label-text">{{ t('config.dashboardUser') }}</span>
            </label>
            <input
              v-model="formData.common.adminUser"
              type="text"
              class="input input-bordered"
              placeholder="admin"
            />
          </div>
          <div class="form-control">
            <label class="label">
              <span class="label-text">{{ t('config.dashboardPassword') }}</span>
            </label>
            <input
              v-model="formData.common.adminPassword"
              type="password"
              class="input input-bordered"
              placeholder="admin"
            />
            <label class="label">
              <span class="label-text-alt text-base-content/50">{{ t('config.dashboardAuthHint') }}</span>
            </label>
          </div>
        </div>

        <!-- 代理配置 -->
        <div class="divider">{{ t('config.proxyConfig') }}</div>

        <!-- 添加/编辑代理表单 -->
        <div class="bg-base-200 rounded-lg p-4 mb-4" :class="{ 'border border-primary': editingProxyIndex >= 0 }">
          <h4 class="font-medium mb-3">
            {{ editingProxyIndex >= 0 ? t('config.editProxy') : t('config.addProxy') }}
          </h4>
          <div class="flex flex-wrap items-end gap-3">
            <div class="form-control">
              <label class="label py-1">
                <span class="label-text text-xs">{{ t('config.name') }}</span>
              </label>
              <input
                v-model="newProxy.name"
                type="text"
                class="input input-bordered input-sm w-32"
                :placeholder="t('config.proxyNamePlaceholder')"
              />
            </div>
            <div class="form-control">
              <label class="label py-1">
                <span class="label-text text-xs">{{ t('config.type') }}</span>
              </label>
              <select v-model="newProxy.type" class="select select-bordered select-sm w-24">
                <option value="tcp">TCP</option>
                <option value="udp">UDP</option>
                <option value="http">HTTP</option>
                <option value="https">HTTPS</option>
                <option value="stcp">STCP</option>
              </select>
            </div>
            <div class="form-control">
              <label class="label py-1">
                <span class="label-text text-xs">{{ t('config.localIP') }}</span>
              </label>
              <input
                v-model="newProxy.localIP"
                type="text"
                class="input input-bordered input-sm w-28"
              />
            </div>
            <div class="form-control">
              <label class="label py-1">
                <span class="label-text text-xs">{{ t('config.localPort') }}</span>
              </label>
              <input
                v-model.number="newProxy.localPort"
                type="number"
                class="input input-bordered input-sm w-20"
                :placeholder="t('config.proxyLocalPortPlaceholder')"
              />
            </div>
            <div v-if="['tcp', 'udp'].includes(newProxy.type)" class="form-control">
              <label class="label py-1">
                <span class="label-text text-xs">{{ t('config.remotePort') }}</span>
              </label>
              <input
                v-model.number="newProxy.remotePort"
                type="number"
                class="input input-bordered input-sm w-24"
                :placeholder="t('config.remotePortPlaceholder')"
              />
            </div>
            <div v-if="['http', 'https'].includes(newProxy.type)" class="form-control">
              <label class="label py-1">
                <span class="label-text text-xs">{{ t('config.subdomain') }}</span>
              </label>
              <input
                v-model="newProxy.subdomain"
                type="text"
                class="input input-bordered input-sm w-32"
                :placeholder="t('config.subdomainPlaceholder')"
              />
            </div>
            <button class="btn btn-primary btn-sm" @click="handleAddProxy">
              <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
              </svg>
              {{ editingProxyIndex >= 0 ? t('config.updateBtn') : t('config.addBtn') }}
            </button>
            <button v-if="editingProxyIndex >= 0" class="btn btn-ghost btn-sm" @click="handleCancelEditProxy">
              {{ t('common.cancel') }}
            </button>
          </div>
        </div>

        <!-- 代理列表 -->
        <div v-if="formData.proxies.length > 0" class="overflow-x-auto">
          <table class="table table-zebra">
            <thead>
              <tr>
                <th>{{ t('config.name') }}</th>
                <th>{{ t('config.type') }}</th>
                <th>{{ t('config.localAddr') }}</th>
                <th>{{ t('config.remoteConfig') }}</th>
                <th class="w-20">{{ t('config.actions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(proxy, index) in formData.proxies" :key="index">
                <td class="font-medium">{{ proxy.name }}</td>
                <td><span class="badge badge-ghost">{{ proxy.type }}</span></td>
                <td>{{ proxy.localIP }}:{{ proxy.localPort }}</td>
                <td>
                  <template v-if="['tcp', 'udp'].includes(proxy.type)">
                    {{ t('config.remotePort') }}: {{ proxy.remotePort }}
                  </template>
                  <template v-else-if="['http', 'https'].includes(proxy.type)">
                    {{ t('config.subdomain') }}: {{ proxy.subdomain || '-' }}
                  </template>
                  <template v-else>-</template>
                </td>
                <td>
                  <div class="flex gap-1">
                    <button class="btn btn-ghost btn-xs" @click="handleEditProxy(index)">
                      <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                      </svg>
                    </button>
                    <button class="btn btn-ghost btn-xs text-error" @click="handleRemoveProxy(index)">
                      <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="text-base-content/50 text-center py-8 bg-base-200 rounded-lg">
          <svg class="h-10 w-10 mx-auto mb-2 opacity-50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
          <p>{{ t('config.noProxyConfig') }}</p>
          <p class="text-sm mt-1">{{ t('config.addProxyAbove') }}</p>
        </div>
      </div>
      <div v-else class="flex-1 flex flex-col items-center justify-center text-base-content/50">
        <svg class="h-16 w-16 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
        </svg>
        <p class="text-lg">{{ t('config.selectConfig') }}</p>
        <p class="text-sm mt-2">{{ t('config.orCreateNew') }}</p>
      </div>
    </div>

    <!-- 删除确认对话框 -->
    <dialog :class="['modal', { 'modal-open': showDeleteDialog }]">
      <div class="modal-box">
        <h3 class="font-bold text-lg">{{ t('config.confirmDeleteTitle') }}</h3>
        <p class="py-4">
          {{ t('config.deleteConfirmMsg', { name: deleteTargetName }) }}
        </p>
        <div class="modal-action">
          <button class="btn btn-ghost" @click="showDeleteDialog = false">{{ t('common.cancel') }}</button>
          <button class="btn btn-error" @click="handleDeleteConfirm">{{ t('common.delete') }}</button>
        </div>
      </div>
      <div class="modal-backdrop" @click="showDeleteDialog = false"></div>
    </dialog>

    <!-- 导入对话框 -->
    <dialog :class="['modal', { 'modal-open': showImportDialog }]">
      <div class="modal-box max-w-3xl">
        <h3 class="font-bold text-lg mb-4">{{ t('config.importConfigFile') }}</h3>

        <div class="grid grid-cols-2 gap-4 mb-4">
          <div class="form-control">
            <label class="label">
              <span class="label-text font-medium">{{ t('config.configNameRequired') }} <span class="text-error">*</span></span>
            </label>
            <input
              v-model="importName"
              type="text"
              class="input input-bordered"
              :placeholder="t('config.enterConfigName')"
            />
          </div>
          <div class="form-control">
            <label class="label">
              <span class="label-text font-medium">{{ t('config.fileFormat') }}</span>
            </label>
            <select v-model="importFormat" class="select select-bordered">
              <option value="toml">{{ t('config.versionNew') }}</option>
              <option value="ini">{{ t('config.versionOld') }}</option>
              <option value="yaml">YAML</option>
            </select>
          </div>
        </div>

        <div class="form-control mb-4">
          <label class="label">
            <span class="label-text font-medium">{{ t('config.configContent') }} <span class="text-error">*</span></span>
          </label>
          <textarea
            v-model="importContent"
            class="textarea textarea-bordered h-64 font-mono text-sm"
            :placeholder="t('config.pasteContent')"
          ></textarea>
        </div>

        <div class="modal-action">
          <button class="btn btn-ghost" @click="showImportDialog = false">{{ t('common.cancel') }}</button>
          <button
            class="btn btn-primary"
            @click="handleImport"
            :disabled="!importName.trim() || !importContent.trim()"
          >
            {{ t('config.importBtn') }}
          </button>
        </div>
      </div>
      <div class="modal-backdrop" @click="showImportDialog = false"></div>
    </dialog>
  </div>
</template>
