import { ref } from 'vue'
import { defineStore } from 'pinia'

export const useThemeStore = defineStore('theme', () => {
  // State
  const theme = ref('light')

  // Actions
  function init() {
    const saved = localStorage.getItem('theme')
    if (saved && (saved === 'light' || saved === 'dark')) {
      theme.value = saved
      applyTheme(saved)
      return
    }

    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
    const systemTheme = prefersDark ? 'dark' : 'light'
    theme.value = systemTheme
    applyTheme(systemTheme)
  }

  function toggle() {
    const newTheme = theme.value === 'light' ? 'dark' : 'light'
    theme.value = newTheme
    applyTheme(newTheme)
    localStorage.setItem('theme', newTheme)
  }

  function setTheme(newTheme) {
    theme.value = newTheme
    applyTheme(newTheme)
    localStorage.setItem('theme', newTheme)
  }

  function applyTheme(newTheme) {
    const html = document.documentElement
    html.setAttribute('data-theme', newTheme)
    html.classList.remove('light', 'dark')
    html.classList.add(newTheme)
    html.style.colorScheme = newTheme
  }

  return {
    theme,
    init,
    toggle,
    setTheme
  }
})
