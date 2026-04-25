import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { versionApi } from '@/api/wails'
import { Events } from '@wailsio/runtime'

export const useVersionStore = defineStore('version', () => {
  // State
  const remoteVersions = ref([])
  const localVersions = ref([])
  const activeVersion = ref('')
  const isLoading = ref(false)
  const downloading = ref(null)
  const downloadProgress = ref(null)
  const error = ref(null)

  // Getters
  const hasActiveVersion = computed(() => !!activeVersion.value)
  const localVersionCount = computed(() => localVersions.value.length)

  // Actions
  async function loadRemoteVersions() {
    isLoading.value = true
    error.value = null
    try {
      const versions = await versionApi.listRemote()
      remoteVersions.value = versions || []

      // 标记已安装状态
      const localMap = new Map(localVersions.value.map(v => [v.version, true]))
      for (const v of remoteVersions.value) {
        if (v && v.version) {
          v.isLocal = localMap.has(v.version)
          v.isActive = v.version === activeVersion.value
        }
      }
    } catch (err) {
      error.value = err.message
      console.error('获取远程版本列表失败:', err)
    } finally {
      isLoading.value = false
    }
  }

  async function loadLocalVersions() {
    try {
      const versions = await versionApi.listLocal()
      localVersions.value = versions || []
    } catch (err) {
      console.error('获取本地版本列表失败:', err)
      localVersions.value = []
    }
  }

  async function loadActiveVersion() {
    try {
      activeVersion.value = await versionApi.getActive()
    } catch (err) {
      console.error('获取活动版本失败:', err)
      activeVersion.value = ''
    }
  }

  async function download(version) {
    downloading.value = version
    downloadProgress.value = { version, percent: 0, downloaded: 0, total: 0, speed: 0 }
    error.value = null

    // 监听下载进度事件
    const cancelListener = Events.On('download-progress', (event) => {
      const data = event.data
      if (data.version === version) {
        downloadProgress.value = data
      }
    })

    try {
      await versionApi.download(version)
      await loadLocalVersions()
      // 更新远程列表中的状态
      const found = remoteVersions.value.find(v => v && v.version === version)
      if (found) {
        found.isLocal = true
      }
      return true
    } catch (err) {
      error.value = err.message
      console.error('下载版本失败:', err)
      throw err
    } finally {
      cancelListener()
      downloading.value = null
      downloadProgress.value = null
    }
  }

  async function setActive(version) {
    isLoading.value = true
    error.value = null
    try {
      await versionApi.setActive(version)
      activeVersion.value = version
      // 更新本地版本列表中的活动状态
      for (const v of localVersions.value) {
        if (v) {
          v.isActive = v.version === version
        }
      }
      // 更新远程版本列表
      for (const v of remoteVersions.value) {
        if (v) {
          v.isActive = v.version === version
        }
      }
      return true
    } catch (err) {
      error.value = err.message
      console.error('设置活动版本失败:', err)
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function deleteVersion(version) {
    isLoading.value = true
    error.value = null
    try {
      await versionApi.delete(version)
      await loadLocalVersions()
      // 更新远程列表
      const found = remoteVersions.value.find(v => v && v.version === version)
      if (found) {
        found.isLocal = false
      }
      if (activeVersion.value === version) {
        activeVersion.value = ''
      }
      return true
    } catch (err) {
      error.value = err.message
      console.error('删除版本失败:', err)
      throw err
    } finally {
      isLoading.value = false
    }
  }

  async function init() {
    await Promise.all([
      loadLocalVersions(),
      loadActiveVersion(),
    ])
  }

  return {
    remoteVersions,
    localVersions,
    activeVersion,
    isLoading,
    downloading,
    downloadProgress,
    error,
    hasActiveVersion,
    localVersionCount,
    loadRemoteVersions,
    loadLocalVersions,
    loadActiveVersion,
    download,
    setActive,
    deleteVersion,
    init,
  }
})
