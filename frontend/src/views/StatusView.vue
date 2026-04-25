<script setup>
import { onMounted, onUnmounted, computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useConfigStore } from '@/stores/config'
import { useProcessStore } from '@/stores/process'
import { useVersionStore } from '@/stores/version'

const { t } = useI18n()
const configStore = useConfigStore()
const processStore = useProcessStore()
const versionStore = useVersionStore()

let refreshInterval = null

// 展开状态
const expandedConfigs = ref(new Set())

onMounted(async () => {
  await configStore.load()
  await processStore.refreshStatus()
  await versionStore.init()

  // 每 3 秒刷新一次状态
  refreshInterval = setInterval(() => {
    processStore.refreshStatus()
  }, 3000)
})

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
})

const runningProcesses = computed(() => {
  return configStore.configs.filter(config => {
    const status = processStore.getStatus(config.id)
    return status?.status === 'running'
  })
})

const stoppedProcesses = computed(() => {
  return configStore.configs.filter(config => {
    const status = processStore.getStatus(config.id)
    return status?.status !== 'running'
  })
})

function formatDuration(startTime) {
  if (!startTime) return '-'
  const start = new Date(startTime)
  const now = new Date()
  const diff = Math.floor((now - start) / 1000)

  if (diff < 0) return '-'

  const hours = Math.floor(diff / 3600)
  const minutes = Math.floor((diff % 3600) / 60)
  const seconds = diff % 60

  if (hours > 0) {
    return `${hours}h ${minutes}m ${seconds}s`
  } else if (minutes > 0) {
    return `${minutes}m ${seconds}s`
  }
  return `${seconds}s`
}

// 格式化流量大小
function formatTraffic(bytes) {
  if (bytes === undefined || bytes === null) return '-'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`
}

// 格式化速度
function formatSpeed(bytesPerSec) {
  if (bytesPerSec === undefined || bytesPerSec === null) return '-'
  if (bytesPerSec < 1024) return `${bytesPerSec} B/s`
  if (bytesPerSec < 1024 * 1024) return `${(bytesPerSec / 1024).toFixed(1)} KB/s`
  return `${(bytesPerSec / 1024 / 1024).toFixed(1)} MB/s`
}

// 获取配置中的代理列表（合并运行时状态）
function getConfigProxies(configId) {
  // 从 configStore 获取配置详情
  const config = configStore.configs.find(c => c.id === configId)
  if (!config || !config.proxies || config.proxies.length === 0) {
    return []
  }

  // 获取运行时状态
  const status = processStore.getStatus(configId)
  const runtimeProxies = status?.proxies || []

  // 合并配置代理和运行时状态
  return config.proxies.map(proxy => {
    // 查找运行时状态
    const runtime = runtimeProxies.find(p => p.name === proxy.name)

    // 构建 HTTP/HTTPS 代理的完整访问链接
    let httpUrl = null
    if (['http', 'https'].includes(proxy.type) && config.common?.subDomainHost && proxy.subdomain) {
      const isHttps = proxy.type === 'https'
      const port = isHttps
        ? (config.common.httpsPort || 443)
        : (config.common.httpPort || 80)
      const protocol = isHttps ? 'https' : 'http'
      const domain = config.common.subDomainHost

      // 端口为默认值时不显示端口
      if ((isHttps && port === 443) || (!isHttps && port === 80)) {
        httpUrl = `${protocol}://${proxy.subdomain}.${domain}`
      } else {
        httpUrl = `${protocol}://${proxy.subdomain}.${domain}:${port}`
      }
    }

    return {
      name: proxy.name,
      type: proxy.type,
      localAddr: proxy.localIP && proxy.localPort ? `${proxy.localIP}:${proxy.localPort}` : '-',
      remoteAddr: proxy.remotePort ? `:${proxy.remotePort}` : '-',
      subdomain: proxy.subdomain,
      httpUrl: httpUrl,
      // 状态：如果进程在运行，使用运行时状态；否则显示未启动
      status: status?.status === 'running' ? (runtime?.status || 'starting') : 'offline',
      // 流量和速度数据
      todayTrafficIn: runtime?.todayTrafficIn || 0,
      todayTrafficOut: runtime?.todayTrafficOut || 0,
      currentSpeedIn: runtime?.currentSpeedIn || 0,
      currentSpeedOut: runtime?.currentSpeedOut || 0,
    }
  })
}

// 切换展开状态
function toggleExpand(configId) {
  if (expandedConfigs.value.has(configId)) {
    expandedConfigs.value.delete(configId)
  } else {
    expandedConfigs.value.add(configId)
  }
}

// 检查是否展开
function isExpanded(configId) {
  return expandedConfigs.value.has(configId)
}

async function handleStart(configId) {
  try {
    await processStore.start(configId)
  } catch (err) {
    alert(t('status.start') + ' ' + t('common.error').toLowerCase() + ': ' + err.message)
  }
}

async function handleStop(configId) {
  try {
    await processStore.stop(configId)
  } catch (err) {
    alert(t('status.stop') + ' ' + t('common.error').toLowerCase() + ': ' + err.message)
  }
}
</script>

<template>
  <div class="h-full overflow-auto p-6">
    <!-- 系统状态卡片 -->
    <div class="grid grid-cols-4 gap-4 mb-8">
      <div class="card bg-base-200 shadow-sm">
        <div class="card-body p-4">
          <div class="flex items-center justify-between">
            <div>
              <h3 class="text-sm text-base-content/60">{{ t('status.running') }}</h3>
              <p class="text-3xl font-bold text-success mt-1">{{ runningProcesses.length }}</p>
            </div>
            <div class="text-success opacity-20">
              <svg class="h-10 w-10" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </div>
          </div>
        </div>
      </div>

      <div class="card bg-base-200 shadow-sm">
        <div class="card-body p-4">
          <div class="flex items-center justify-between">
            <div>
              <h3 class="text-sm text-base-content/60">{{ t('status.stopped') }}</h3>
              <p class="text-3xl font-bold mt-1">{{ stoppedProcesses.length }}</p>
            </div>
            <div class="text-base-content/20">
              <svg class="h-10 w-10" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 10a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z" />
              </svg>
            </div>
          </div>
        </div>
      </div>

      <div class="card bg-base-200 shadow-sm">
        <div class="card-body p-4">
          <div class="flex items-center justify-between">
            <div>
              <h3 class="text-sm text-base-content/60">{{ t('status.totalConfigs') }}</h3>
              <p class="text-3xl font-bold mt-1">{{ configStore.configs.length }}</p>
            </div>
            <div class="text-base-content/20">
              <svg class="h-10 w-10" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
            </div>
          </div>
        </div>
      </div>

      <div class="card bg-base-200 shadow-sm">
        <div class="card-body p-4">
          <div class="flex items-center justify-between">
            <div>
              <h3 class="text-sm text-base-content/60">{{ t('status.frpcVersion') }}</h3>
              <p class="text-xl font-bold mt-1 truncate">{{ versionStore.activeVersion || t('status.notSet') }}</p>
            </div>
            <div :class="versionStore.hasActiveVersion ? 'text-primary opacity-20' : 'text-warning opacity-40'">
              <svg class="h-10 w-10" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
              </svg>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 警告提示 -->
    <div v-if="!versionStore.hasActiveVersion" class="alert alert-warning mb-6">
      <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
      </svg>
      <div>
        <h3 class="font-bold">{{ t('status.noActiveVersion') }}</h3>
        <p class="text-sm">{{ t('status.pleaseSetVersion') }}</p>
      </div>
    </div>

    <!-- 进程列表 -->
    <h2 class="text-lg font-bold mb-4">{{ t('status.processStatus') }}</h2>

    <div v-if="configStore.configs.length === 0" class="text-center py-16 text-base-content/50 bg-base-200 rounded-lg">
      <svg class="h-16 w-16 mx-auto mb-4 opacity-50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
      </svg>
      <p class="text-lg">{{ t('status.noConfigs') }}</p>
      <p class="text-sm mt-2">{{ t('status.pleaseCreateConfig') }}</p>
    </div>

    <div v-else class="space-y-3">
      <div
        v-for="config in configStore.configs"
        :key="config.id"
        class="card bg-base-200 shadow-sm"
      >
        <div class="card-body p-4">
          <!-- 主状态行 -->
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-4 flex-1 cursor-pointer" @click="toggleExpand(config.id)">
              <!-- 展开箭头 -->
              <svg
                :class="['h-4 w-4 transition-transform', isExpanded(config.id) ? 'rotate-90' : '']"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
              </svg>
              <div
                :class="[
                  'w-3 h-3 rounded-full transition-colors',
                  processStore.getStatus(config.id)?.status === 'running' ? 'bg-success animate-pulse' : 'bg-base-content/30'
                ]"
              ></div>
              <div>
                <h3 class="font-medium">{{ config.name }}</h3>
                <p class="text-sm text-base-content/60">{{ config.description || '-' }}</p>
              </div>
            </div>
            <div class="flex items-center gap-6">
              <div v-if="processStore.getStatus(config.id)?.status === 'running'" class="text-sm text-base-content/60">
                <span class="text-success">{{ t('status.running') }}</span>
                <span class="mx-2">·</span>
                <span>{{ formatDuration(processStore.getStatus(config.id)?.startedAt) }}</span>
                <!-- 显示总流量 -->
                <template v-if="processStore.getStatus(config.id)?.totalTrafficIn !== undefined">
                  <span class="mx-2">·</span>
                  <span class="text-info">↓{{ formatTraffic(processStore.getStatus(config.id)?.totalTrafficIn) }}</span>
                  <span class="text-warning ml-1">↑{{ formatTraffic(processStore.getStatus(config.id)?.totalTrafficOut) }}</span>
                </template>
              </div>
              <div v-else class="text-sm text-base-content/50">
                {{ t('status.stopped') }}
              </div>
              <div class="flex gap-2">
                <button
                  v-if="processStore.getStatus(config.id)?.status === 'running'"
                  class="btn btn-error btn-sm"
                  @click.stop="handleStop(config.id)"
                >
                  <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 10a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z" />
                  </svg>
                  {{ t('status.stop') }}
                </button>
                <button
                  v-else
                  class="btn btn-success btn-sm"
                  @click.stop="handleStart(config.id)"
                  :disabled="!versionStore.hasActiveVersion"
                >
                  <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  {{ t('status.start') }}
                </button>
              </div>
            </div>
          </div>

          <!-- 展开的代理详情 -->
          <div v-if="isExpanded(config.id)" class="mt-4 pt-4 border-t border-base-300">
            <h4 class="text-sm font-medium mb-3 text-base-content/70">{{ t('status.proxyStatus') }}</h4>

            <!-- 代理列表 -->
            <div v-if="getConfigProxies(config.id).length > 0" class="space-y-2">
              <div
                v-for="proxy in getConfigProxies(config.id)"
                :key="proxy.name"
                class="flex items-center justify-between p-3 bg-base-300/50 rounded-lg"
              >
                <div class="flex items-center gap-3">
                  <div
                    :class="[
                      'w-2 h-2 rounded-full',
                      proxy.status === 'running' ? 'bg-success animate-pulse' :
                      proxy.status === 'error' ? 'bg-error' :
                      proxy.status === 'starting' ? 'bg-warning animate-pulse' : 'bg-base-content/30'
                    ]"
                  ></div>
                  <div>
                    <span class="font-medium text-sm">{{ proxy.name }}</span>
                    <span class="text-xs text-base-content/50 ml-2">{{ proxy.type?.toUpperCase() }}</span>
                  </div>
                </div>
                <div class="flex items-center gap-4 text-xs text-base-content/60">
                  <!-- 本地地址 -->
                  <div v-if="proxy.localAddr !== '-'" class="flex items-center gap-1">
                    <span class="text-base-content/40">{{ t('status.local') }}:</span>
                    <span>{{ proxy.localAddr }}</span>
                  </div>
                  <!-- HTTP/HTTPS 代理显示访问链接 -->
                  <div v-if="proxy.httpUrl" class="flex items-center gap-1">
                    <a
                      :href="proxy.httpUrl"
                      target="_blank"
                      class="text-primary hover:underline flex items-center gap-1"
                      :title="t('status.openInBrowser')"
                    >
                      <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                      </svg>
                      {{ proxy.httpUrl }}
                    </a>
                  </div>
                  <!-- TCP/UDP 远程端口 -->
                  <div v-if="proxy.remoteAddr !== '-' && !proxy.httpUrl" class="flex items-center gap-1">
                    <span class="text-base-content/40">{{ t('status.remote') }}:</span>
                    <span>{{ proxy.remoteAddr }}</span>
                  </div>
                  <!-- 流量统计（运行时始终显示） -->
                  <template v-if="processStore.getStatus(config.id)?.status === 'running'">
                    <div class="flex items-center gap-1">
                      <span class="text-info">↓{{ formatTraffic(proxy.todayTrafficIn) }}</span>
                      <span class="text-warning ml-2">↑{{ formatTraffic(proxy.todayTrafficOut) }}</span>
                    </div>
                    <div class="flex items-center gap-1">
                      <svg class="h-3 w-3 text-info" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 14l-7 7m0 0l-7-7m7 7V3" />
                      </svg>
                      <span>{{ formatSpeed(proxy.currentSpeedIn) }}</span>
                      <svg class="h-3 w-3 text-warning ml-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 10l7-7m0 0l7 7m-7-7v18" />
                      </svg>
                      <span>{{ formatSpeed(proxy.currentSpeedOut) }}</span>
                    </div>
                  </template>
                  <!-- 状态标签 -->
                  <div v-else-if="processStore.getStatus(config.id)?.status !== 'running'" class="text-base-content/40">
                    {{ t('status.notStarted') }}
                  </div>
                </div>
              </div>
            </div>

            <!-- 无代理配置 -->
            <div v-else class="text-center py-4 text-base-content/50 text-sm">
              <p>{{ t('status.noProxyConfig') }}</p>
            </div>

            <!-- 总流量统计（运行时始终显示） -->
            <div v-if="processStore.getStatus(config.id)?.status === 'running'" class="mt-4 flex justify-between text-sm">
              <div>
                <span class="text-base-content/60">{{ t('status.totalDownload') }}: </span>
                <span class="text-info font-medium">{{ formatTraffic(processStore.getStatus(config.id)?.totalTrafficIn) }}</span>
              </div>
              <div>
                <span class="text-base-content/60">{{ t('status.totalUpload') }}: </span>
                <span class="text-warning font-medium">{{ formatTraffic(processStore.getStatus(config.id)?.totalTrafficOut) }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
