import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { configApi } from '@/api/wails'

export const useConfigStore = defineStore('config', () => {
  // State
  const configs = ref([]) // 完整配置列表（包含 proxies）
  const isLoading = ref(false)
  const error = ref(null)

  // Getters
  const configCount = computed(() => configs.value.length)

  // Actions
  async function load() {
    isLoading.value = true
    error.value = null
    try {
      // 获取配置列表
      const list = await configApi.list()
      if (!list || list.length === 0) {
        configs.value = []
        return
      }

      // 获取每个配置的完整信息（包含 proxies）
      const fullConfigs = await Promise.all(
        list.map(async (meta) => {
          try {
            const fullConfig = await configApi.get(meta.id)
            return fullConfig || meta
          } catch (err) {
            console.error('获取配置详情失败:', meta.id, err)
            return meta
          }
        })
      )
      configs.value = fullConfigs
    } catch (err) {
      error.value = err.message
      console.error('加载配置列表失败:', err)
    } finally {
      isLoading.value = false
    }
  }

  async function get(id) {
    try {
      return await configApi.get(id)
    } catch (err) {
      console.error('获取配置失败:', err)
      return null
    }
  }

  async function save(config) {
    try {
      await configApi.save(config)
      await load()
      return true
    } catch (err) {
      console.error('保存配置失败:', err)
      throw err
    }
  }

  async function deleteConfig(id) {
    try {
      await configApi.delete(id)
      await load()
      return true
    } catch (err) {
      console.error('删除配置失败:', err)
      throw err
    }
  }

  async function create(name) {
    try {
      const config = await configApi.new(name)
      if (config) {
        await load()
        return config
      }
      return null
    } catch (err) {
      console.error('创建配置失败:', err)
      throw err
    }
  }

  async function importConfig(name, content, format) {
    try {
      const config = await configApi.import(name, content, format)
      if (config) {
        await load()
        return config
      }
      return null
    } catch (err) {
      console.error('导入配置失败:', err)
      throw err
    }
  }

  async function exportConfig(id, format) {
    try {
      return await configApi.export(id, format)
    } catch (err) {
      console.error('导出配置失败:', err)
      throw err
    }
  }

  return {
    configs,
    isLoading,
    error,
    configCount,
    load,
    get,
    save,
    deleteConfig,
    create,
    importConfig,
    exportConfig,
  }
})
