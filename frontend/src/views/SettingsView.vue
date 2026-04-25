<script setup>
import { ref, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from '@/stores/settings'
import { setLocale } from '@/i18n'

const { t } = useI18n()
const settingsStore = useSettingsStore()
const formData = ref({
  theme: 'system',
  language: 'zh-CN',
  autoRestart: false,
  autoStart: false,
  minimizeToTray: true,
})

const isSaving = ref(false)
const saveMessage = ref('')

onMounted(async () => {
  await settingsStore.load()
  formData.value = { ...settingsStore.settings }
})

// 应用主题
function applyTheme(theme) {
  const html = document.documentElement

  if (theme === 'system') {
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
    html.setAttribute('data-theme', prefersDark ? 'dark' : 'light')
  } else {
    html.setAttribute('data-theme', theme)
  }
}

// 监听主题变化 - 立即应用
watch(() => formData.value.theme, (newTheme) => {
  applyTheme(newTheme)
})

// 监听语言变化 - 立即应用
watch(() => formData.value.language, (newLang) => {
  setLocale(newLang)
})

async function handleSave() {
  isSaving.value = true
  saveMessage.value = ''

  try {
    await settingsStore.save(formData.value)
    applyTheme(formData.value.theme)
    saveMessage.value = t('settings.saved')
    setTimeout(() => { saveMessage.value = '' }, 2000)
  } catch (err) {
    alert(t('common.error') + ': ' + err.message)
  } finally {
    isSaving.value = false
  }
}

function handleReset() {
  formData.value = {
    theme: 'system',
    language: 'zh-CN',
    autoRestart: false,
    autoStart: false,
    minimizeToTray: true,
  }
}

const themeOptions = [
  { value: 'light', label: 'themeLight', icon: '☀️' },
  { value: 'dark', label: 'themeDark', icon: '🌙' },
  { value: 'system', label: 'themeSystem', icon: '💻' },
]
</script>

<template>
  <div class="h-full overflow-auto p-6">
    <div class="max-w-2xl">
      <div class="flex items-center justify-between mb-6">
        <h2 class="text-xl font-bold">{{ t('settings.title') }}</h2>
        <span v-if="saveMessage" class="badge badge-success">{{ saveMessage }}</span>
      </div>

      <!-- 外观设置 -->
      <div class="card bg-base-200 mb-6">
        <div class="card-body">
          <h3 class="font-medium mb-4">{{ t('settings.appearance') }}</h3>

          <div class="form-control mb-4">
            <label class="label">
              <span class="label-text font-medium">{{ t('settings.theme') }}</span>
            </label>
            <div class="flex gap-3 mt-2">
              <button
                v-for="opt in themeOptions"
                :key="opt.value"
                :class="[
                  'btn btn-outline flex-1',
                  formData.theme === opt.value ? 'btn-primary' : ''
                ]"
                @click="formData.theme = opt.value"
              >
                <span class="mr-2">{{ opt.icon }}</span>
                {{ t('settings.' + opt.label) }}
              </button>
            </div>
          </div>

          <div class="form-control">
            <label class="label">
              <span class="label-text font-medium">{{ t('settings.language') }}</span>
            </label>
            <select v-model="formData.language" class="select select-bordered max-w-xs">
              <option value="zh-CN">简体中文</option>
              <option value="en-US">English</option>
            </select>
          </div>
        </div>
      </div>

      <!-- 行为设置 -->
      <div class="card bg-base-200 mb-6">
        <div class="card-body">
          <h3 class="font-medium mb-4">{{ t('settings.behavior') }}</h3>

          <div class="space-y-4">
            <div class="flex items-center justify-between">
              <div>
                <p class="font-medium">{{ t('settings.autoStart') }}</p>
                <p class="text-sm text-base-content/60">{{ t('settings.autoStartDesc') }}</p>
              </div>
              <input v-model="formData.autoStart" type="checkbox" class="toggle toggle-primary" />
            </div>

            <div class="divider my-2"></div>

            <div class="flex items-center justify-between">
              <div>
                <p class="font-medium">{{ t('settings.autoRestart') }}</p>
                <p class="text-sm text-base-content/60">{{ t('settings.autoRestartDesc') }}</p>
              </div>
              <input v-model="formData.autoRestart" type="checkbox" class="toggle toggle-primary" />
            </div>

            <div class="divider my-2"></div>

            <div class="flex items-center justify-between">
              <div>
                <p class="font-medium">{{ t('settings.minimizeToTray') }}</p>
                <p class="text-sm text-base-content/60">{{ t('settings.minimizeToTrayDesc') }}</p>
              </div>
              <input v-model="formData.minimizeToTray" type="checkbox" class="toggle toggle-primary" />
            </div>
          </div>
        </div>
      </div>

      <!-- 操作按钮 -->
      <div class="flex gap-3">
        <button class="btn btn-primary" @click="handleSave" :disabled="isSaving">
          <span v-if="isSaving" class="loading loading-spinner loading-sm"></span>
          {{ t('common.save') }}{{ isSaving ? '...' : '' }}
        </button>
        <button class="btn btn-ghost" @click="handleReset">
          {{ t('common.reset') }}
        </button>
      </div>

      <!-- 关于信息 -->
      <div class="divider"></div>
      <div class="bg-base-200 rounded-lg p-4">
        <h3 class="font-medium mb-2">{{ t('settings.about') }}</h3>
        <div class="text-sm text-base-content/60 space-y-1">
          <p><span class="font-medium text-base-content">Magic FRPc</span> v0.1.0</p>
          <p>{{ t('settings.aboutDesc') }}</p>
          <p class="mt-2">
            <a href="https://github.com/fatedier/frp" target="_blank" class="link link-primary">
              {{ t('settings.frpOfficial') }}
            </a>
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
