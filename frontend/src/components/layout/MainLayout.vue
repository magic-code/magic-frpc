<script setup>
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const navItems = [
  { id: 'configs', labelKey: 'nav.configs', path: '/configs' },
  { id: 'status', labelKey: 'nav.status', path: '/status' },
  { id: 'version', labelKey: 'nav.versions', path: '/version' },
  { id: 'logs', labelKey: 'nav.logs', path: '/logs' },
  { id: 'settings', labelKey: 'nav.settings', path: '/settings' },
]

function navigateTo(path) {
  router.push(path)
}
</script>

<template>
  <div class="flex h-screen w-screen">
    <!-- 侧边栏 -->
    <aside class="bg-base-200 flex h-full w-56 flex-col border-r border-base-300">
      <!-- Logo -->
      <div class="flex h-14 items-center gap-2 border-b border-base-300 px-4">
        <svg class="h-6 w-6 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
        </svg>
        <span class="text-lg font-bold text-primary">Magic FRPc</span>
      </div>

      <!-- 导航菜单 -->
      <nav class="flex-1 p-2">
        <ul class="menu menu-lg gap-1">
          <li v-for="item in navItems" :key="item.id">
            <button
              type="button"
              class="flex items-center gap-3 rounded-lg px-3 py-3 text-left transition-colors w-full"
              :class="route.path === item.path ? 'bg-primary text-primary-content' : 'hover:bg-base-300'"
              @click="navigateTo(item.path)"
            >
              <span class="font-medium">{{ t(item.labelKey) }}</span>
            </button>
          </li>
        </ul>
      </nav>

      <!-- 底部信息 -->
      <div class="border-t border-base-300 p-4">
        <p class="text-xs text-base-content/50">v0.1.0</p>
      </div>
    </aside>

    <!-- 主内容区 -->
    <div class="flex flex-1 flex-col overflow-hidden">
      <!-- 顶部栏 -->
      <header class="flex h-14 items-center justify-between border-b border-base-300 bg-base-100 px-4">
        <h1 class="text-lg font-semibold">Magic FRPc</h1>
        <div class="flex items-center gap-2">
          <button class="btn btn-ghost btn-sm">
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v2.25m6.364.386l-1.591 1.591M21 12h-2.25m-.386 6.364l-1.591-1.591M12 18.75V21m-4.773-4.227l-1.591 1.591M5.25 12H3m4.227-4.773L5.636 5.636M15.75 12a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0z" />
            </svg>
          </button>
        </div>
      </header>

      <!-- 内容区域 -->
      <main class="flex-1 overflow-auto bg-base-100">
        <slot />
      </main>
    </div>
  </div>
</template>
