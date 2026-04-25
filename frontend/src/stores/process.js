import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { frpcApi } from '@/api/wails'

export const useProcessStore = defineStore('process', () => {
  // State
  const processes = ref({}) // configId -> ProcessState
  const logs = ref({}) // configId -> LogEntry[]
  const isLoading = ref(false)

  // Getters
  const runningCount = computed(() => {
    return Object.values(processes.value).filter(p => p?.status === 'running').length
  })

  // Actions
  async function refreshStatus() {
    try {
      const status = await frpcApi.getAllStatus()
      if (status) {
        // 处理返回的数据
        const result = {}
        for (const [key, value] of Object.entries(status)) {
          result[key] = value || { configId: key, status: 'stopped' }
        }
        processes.value = result
      }
    } catch (err) {
      console.error('获取进程状态失败:', err)
    }
  }

  async function start(configId) {
    isLoading.value = true
    try {
      await frpcApi.start(configId)
      await refreshStatus()
      return true
    } catch (err) {
      console.error('启动进程失败:', err)
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function stop(configId) {
    isLoading.value = true
    try {
      await frpcApi.stop(configId)
      await refreshStatus()
      return true
    } catch (err) {
      console.error('停止进程失败:', err)
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function restart(configId) {
    isLoading.value = true
    try {
      await frpcApi.restart(configId)
      await refreshStatus()
      return true
    } catch (err) {
      console.error('重启进程失败:', err)
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function loadLogs(configId, limit = 200) {
    try {
      const entries = await frpcApi.getLogs(configId, limit)
      logs.value[configId] = entries || []
    } catch (err) {
      console.error('获取日志失败:', err)
    }
  }

  async function clearLogs(configId) {
    try {
      await frpcApi.clearLogs(configId)
      logs.value[configId] = []
    } catch (err) {
      console.error('清除日志失败:', err)
    }
  }

  function getStatus(configId) {
    return processes.value[configId] || { configId, status: 'stopped' }
  }

  function getLogs(configId) {
    return logs.value[configId] || []
  }

  return {
    processes,
    logs,
    isLoading,
    runningCount,
    refreshStatus,
    start,
    stop,
    restart,
    loadLogs,
    clearLogs,
    getStatus,
    getLogs,
  }
})
