<script setup>
import { ref, onMounted, onUnmounted, computed, watch, nextTick } from 'vue'
import { useConfigStore } from '@/stores/config'
import { useProcessStore } from '@/stores/process'
import { appApi } from '@/api/wails'

const configStore = useConfigStore()
const processStore = useProcessStore()

const selectedConfigId = ref('app')
const logs = ref([])
const autoScroll = ref(true)
const logContainer = ref(null)
const selectedLevel = ref('all')

let refreshInterval = null

const levelOptions = [
  { value: 'all', label: '全部' },
  { value: 'info', label: 'Info' },
  { value: 'warn', label: 'Warning' },
  { value: 'error', label: 'Error' },
]

onMounted(async () => {
  await configStore.load()
  await loadLogs()

  refreshInterval = setInterval(() => {
    loadLogs()
  }, 2000)
})

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
})

watch(selectedConfigId, () => {
  logs.value = []
  loadLogs()
})

async function loadLogs() {
  try {
    if (selectedConfigId.value === 'app') {
      const entries = await appApi.getLogs(500)
      logs.value = entries || []
    } else {
      await processStore.loadLogs(selectedConfigId.value, 500)
      logs.value = processStore.getLogs(selectedConfigId.value)
    }

    if (autoScroll.value) {
      await nextTick()
      scrollToBottom()
    }
  } catch (err) {
    console.error('加载日志失败:', err)
  }
}

async function handleClearLogs() {
  try {
    if (selectedConfigId.value === 'app') {
      await appApi.clearLogs()
    } else {
      await processStore.clearLogs(selectedConfigId.value)
    }
    logs.value = []
  } catch (err) {
    console.error('清除日志失败:', err)
  }
}

function scrollToBottom() {
  if (logContainer.value) {
    logContainer.value.scrollTop = logContainer.value.scrollHeight
  }
}

const filteredLogs = computed(() => {
  if (selectedLevel.value === 'all') {
    return logs.value
  }
  return logs.value.filter(log => {
    const level = (log.level || '').toLowerCase()
    if (selectedLevel.value === 'warn') {
      return level === 'warn' || level === 'warning'
    }
    return level === selectedLevel.value.toLowerCase()
  })
})

function formatTime(timestamp) {
  if (!timestamp) return ''
  const date = new Date(timestamp)
  return date.toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

function getLevelClass(level) {
  const l = (level || '').toLowerCase()
  switch (l) {
    case 'error':
      return 'text-error bg-error/10'
    case 'warn':
    case 'warning':
      return 'text-warning bg-warning/10'
    case 'debug':
      return 'text-info bg-info/10'
    default:
      return 'text-base-content bg-base-200'
  }
}

function getLevelBadge(level) {
  const l = (level || '').toLowerCase()
  switch (l) {
    case 'error':
      return 'badge-error'
    case 'warn':
    case 'warning':
      return 'badge-warning'
    case 'debug':
      return 'badge-info'
    default:
      return 'badge-ghost'
  }
}
</script>

<template>
  <div class="flex h-full">
    <!-- 左侧：日志源选择 -->
    <div class="w-56 border-r border-base-300 bg-base-200 flex-shrink-0 flex flex-col">
      <div class="p-3 border-b border-base-300">
        <h2 class="font-semibold">日志源</h2>
      </div>
      <div class="flex-1 overflow-auto p-2">
        <ul class="menu menu-sm">
          <li>
            <button
              :class="{ 'active': selectedConfigId === 'app' }"
              @click="selectedConfigId = 'app'"
            >
              <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
              </svg>
              应用日志
            </button>
          </li>
          <li v-if="configStore.configs.length > 0" class="menu-title">
            <span>进程日志</span>
          </li>
          <li v-for="config in configStore.configs" :key="config.id">
            <button
              :class="{ 'active': selectedConfigId === config.id }"
              @click="selectedConfigId = config.id"
            >
              <div class="flex items-center gap-2 w-full">
                <div
                  :class="[
                    'w-2 h-2 rounded-full flex-shrink-0',
                    processStore.getStatus(config.id)?.status === 'running' ? 'bg-success' : 'bg-base-content/30'
                  ]"
                ></div>
                <span class="truncate">{{ config.name }}</span>
              </div>
            </button>
          </li>
        </ul>
      </div>
    </div>

    <!-- 右侧：日志内容 -->
    <div class="flex-1 flex flex-col min-w-0 overflow-hidden">
      <!-- 工具栏 -->
      <div class="flex items-center justify-between p-3 border-b border-base-300 bg-base-100">
        <div class="flex items-center gap-4">
          <select v-model="selectedLevel" class="select select-bordered select-sm w-32">
            <option v-for="opt in levelOptions" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </option>
          </select>
          <label class="flex items-center gap-2 cursor-pointer">
            <input v-model="autoScroll" type="checkbox" class="toggle toggle-sm toggle-primary" />
            <span class="text-sm">自动滚动</span>
          </label>
        </div>
        <div class="flex items-center gap-3">
          <span class="text-sm text-base-content/60">
            {{ filteredLogs.length }} 条日志
          </span>
          <button class="btn btn-ghost btn-sm" @click="handleClearLogs">
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
            清除
          </button>
        </div>
      </div>

      <!-- 日志内容 -->
      <div
        ref="logContainer"
        class="flex-1 overflow-auto bg-base-300"
      >
        <div v-if="filteredLogs.length === 0" class="flex flex-col items-center justify-center h-full text-base-content/50">
          <svg class="h-12 w-12 mb-4 opacity-50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          <p>暂无日志</p>
        </div>
        <div v-else class="p-2 space-y-1">
          <div
            v-for="(log, index) in filteredLogs"
            :key="index"
            class="flex items-start gap-3 py-2 px-3 rounded hover:bg-base-200 transition-colors"
          >
            <span class="text-xs text-base-content/50 flex-shrink-0 w-20 font-mono">
              {{ formatTime(log.timestamp) }}
            </span>
            <span
              :class="['badge badge-sm flex-shrink-0 w-16 justify-center', getLevelBadge(log.level)]"
            >
              {{ (log.level || 'INFO').toUpperCase() }}
            </span>
            <span class="flex-1 text-sm whitespace-pre-wrap break-all font-mono">
              {{ log.message }}
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
