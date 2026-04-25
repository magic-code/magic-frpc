import { createI18n } from 'vue-i18n'
import zhCN from './locales/zh-CN'
import enUS from './locales/en-US'

// 获取保存的语言设置或使用浏览器语言
function getDefaultLocale() {
  // 尝试从 localStorage 获取
  const saved = localStorage.getItem('settings')
  if (saved) {
    try {
      const settings = JSON.parse(saved)
      if (settings.language) {
        return settings.language
      }
    } catch (e) {
      // ignore
    }
  }

  // 使用浏览器语言
  const browserLang = navigator.language || navigator.userLanguage
  if (browserLang.startsWith('zh')) {
    return 'zh-CN'
  }
  return 'en-US'
}

const i18n = createI18n({
  legacy: false, // 使用 Composition API 模式
  locale: getDefaultLocale(),
  fallbackLocale: 'en-US',
  messages: {
    'zh-CN': zhCN,
    'en-US': enUS,
  },
})

export default i18n

// 切换语言
export function setLocale(locale) {
  i18n.global.locale.value = locale
  document.documentElement.setAttribute('lang', locale)
  localStorage.setItem('locale', locale)
}

// 获取当前语言
export function getLocale() {
  return i18n.global.locale.value
}
