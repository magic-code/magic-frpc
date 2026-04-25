import { ref, watch } from 'vue'
import { defineStore } from 'pinia'
import { settingsApi } from '@/api/wails'
import { setLocale } from '@/i18n'

export const useSettingsStore = defineStore('settings', () => {
  // State
  const settings = ref({
    theme: 'system',
    language: 'zh-CN',
    autoRestart: false,
    autoStart: false,
    minimizeToTray: true,
  })
  const isLoading = ref(false)
  const error = ref(null)

  // Actions
  async function load() {
    isLoading.value = true
    error.value = null
    try {
      const loaded = await settingsApi.get()
      if (loaded) {
        settings.value = {
          theme: loaded.theme || 'system',
          language: loaded.language || 'zh-CN',
          autoRestart: loaded.autoRestart || false,
          autoStart: loaded.autoStart || false,
          minimizeToTray: loaded.minimizeToTray !== false,
        }
        // 应用主题
        applyTheme(settings.value.theme)
        // 应用语言
        setLocale(settings.value.language)
      }
    } catch (err) {
      error.value = err.message
      console.error('加载设置失败:', err)
    } finally {
      isLoading.value = false
    }
  }

  async function save(newSettings) {
    isLoading.value = true
    error.value = null
    try {
      await settingsApi.save(newSettings)
      settings.value = { ...settings.value, ...newSettings }
      return true
    } catch (err) {
      error.value = err.message
      console.error('保存设置失败:', err)
      throw err
    } finally {
      isLoading.value = false
    }
  }

  function applyTheme(theme) {
    const html = document.documentElement
    if (theme === 'system') {
      const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
      html.setAttribute('data-theme', prefersDark ? 'dark' : 'light')
    } else {
      html.setAttribute('data-theme', theme)
    }
  }

  async function updateTheme(theme) {
    const result = await save({ theme })
    if (result) {
      applyTheme(theme)
    }
    return result
  }

  async function updateLanguage(language) {
    const result = await save({ language })
    if (result) {
      setLocale(language)
    }
    return result
  }

  return {
    settings,
    isLoading,
    error,
    load,
    save,
    updateTheme,
    updateLanguage,
    applyTheme,
  }
})
