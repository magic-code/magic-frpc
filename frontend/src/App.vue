<script setup>
import { onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useSettingsStore } from '@/stores/settings'
import MainLayout from './components/layout/MainLayout.vue'

const route = useRoute()
const settingsStore = useSettingsStore()

onMounted(async () => {
  // 加载设置并应用主题
  await settingsStore.load()
})
</script>

<template>
  <div class="h-screen w-screen bg-base-100 text-base-content">
    <MainLayout>
      <router-view />
    </MainLayout>
  </div>
</template>
