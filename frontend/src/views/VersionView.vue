<script setup>
import { ref, onMounted, computed } from 'vue'
import { useVersionStore } from '@/stores/version'

const versionStore = useVersionStore()
const showRemoteDialog = ref(false)
const showDeleteDialog = ref(false)
const deleteTargetVersion = ref('')

onMounted(async () => {
  await versionStore.init()
})

const sortedLocalVersions = computed(() => {
  return [...versionStore.localVersions].sort((a, b) => {
    if (a.isActive) return -1
    if (b.isActive) return 1
    return new Date(b.installedAt) - new Date(a.installedAt)
  })
})

async function openRemoteDialog() {
  showRemoteDialog.value = true
  await versionStore.loadRemoteVersions()
}

async function handleDownload(version) {
  try {
    await versionStore.download(version)
    // 刷新远程版本列表以更新状态
    await versionStore.loadRemoteVersions()
  } catch (err) {
    alert('下载失败: ' + err.message)
  }
}

async function handleSetActive(version) {
  try {
    await versionStore.setActive(version)
  } catch (err) {
    alert('设置失败: ' + err.message)
  }
}

function openDeleteDialog(version) {
  deleteTargetVersion.value = version
  showDeleteDialog.value = true
}

async function handleDeleteConfirm() {
  try {
    await versionStore.deleteVersion(deleteTargetVersion.value)
    showDeleteDialog.value = false
    deleteTargetVersion.value = ''
  } catch (err) {
    alert('删除失败: ' + err.message)
  }
}

function formatSize(bytes) {
  if (!bytes) return '-'
  const mb = bytes / (1024 * 1024)
  if (mb >= 100) {
    return `${(mb / 1024).toFixed(1)} GB`
  }
  return `${mb.toFixed(1)} MB`
}

function formatDate(dateStr) {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

function formatSpeed(bytesPerSec) {
  if (!bytesPerSec) return ''
  const mb = bytesPerSec / (1024 * 1024)
  if (mb >= 1) {
    return `${mb.toFixed(1)} MB/s`
  }
  const kb = bytesPerSec / 1024
  return `${kb.toFixed(0)} KB/s`
}
</script>

<template>
  <div class="h-full overflow-auto p-6">
    <!-- 顶部状态栏 -->
    <div class="flex items-start justify-between mb-6">
      <div>
        <h2 class="text-xl font-bold">版本管理</h2>
        <p class="text-sm text-base-content/60 mt-1">
          下载和管理 frpc 二进制文件版本
        </p>
      </div>
      <button
        class="btn btn-primary"
        @click="openRemoteDialog"
        :disabled="versionStore.isLoading"
      >
        <svg v-if="versionStore.isLoading" class="h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
        <svg v-else class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
        检查更新
      </button>
    </div>

    <!-- 当前版本状态 -->
    <div class="card bg-base-200 mb-6">
      <div class="card-body p-4">
        <h3 class="font-medium mb-2">当前活动版本</h3>
        <div v-if="versionStore.activeVersion" class="flex items-center gap-3">
          <span class="badge badge-success badge-lg">v{{ versionStore.activeVersion }}</span>
          <span class="text-sm text-base-content/60">已激活，可以启动 frpc 进程</span>
        </div>
        <div v-else class="flex items-center gap-3">
          <span class="badge badge-ghost badge-lg">未设置</span>
          <span class="text-sm text-base-content/60">请下载并设置一个版本</span>
        </div>
      </div>
    </div>

    <!-- 本地版本列表 -->
    <h3 class="font-medium mb-4">已安装版本 ({{ versionStore.localVersions.length }})</h3>

    <div v-if="versionStore.localVersions.length === 0" class="text-center py-16 text-base-content/50 bg-base-200 rounded-lg">
      <svg class="h-16 w-16 mx-auto mb-4 opacity-50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
      </svg>
      <p class="text-lg">暂无已安装的版本</p>
      <p class="text-sm mt-2">点击"检查更新"从 GitHub 下载 frpc</p>
    </div>

    <div v-else class="space-y-3">
      <div
        v-for="version in sortedLocalVersions"
        :key="version.version"
        class="card bg-base-200"
      >
        <div class="card-body p-4">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-4">
              <div class="text-2xl font-bold">v{{ version.version }}</div>
              <span v-if="version.isActive" class="badge badge-success">活动</span>
            </div>
            <div class="flex items-center gap-4">
              <span class="text-sm text-base-content/60">
                安装于 {{ formatDate(version.installedAt) }}
              </span>
              <div class="flex gap-2">
                <button
                  v-if="!version.isActive"
                  class="btn btn-ghost btn-sm"
                  @click="handleSetActive(version.version)"
                >
                  设为活动
                </button>
                <button
                  class="btn btn-ghost btn-sm text-error hover:bg-error hover:text-error-content"
                  @click="openDeleteDialog(version.version)"
                  :disabled="version.isActive"
                >
                  <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 远程版本对话框 -->
    <dialog :class="['modal', { 'modal-open': showRemoteDialog }]">
      <div class="modal-box max-w-2xl">
        <div class="flex items-center justify-between mb-4">
          <h3 class="font-bold text-lg">可用版本</h3>
          <button class="btn btn-ghost btn-sm btn-circle" @click="showRemoteDialog = false">
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div v-if="versionStore.isLoading" class="flex flex-col items-center justify-center py-12">
          <span class="loading loading-spinner loading-lg"></span>
          <p class="mt-4 text-base-content/60">正在从 GitHub 获取版本列表...</p>
        </div>

        <div v-else-if="versionStore.remoteVersions.length === 0" class="text-center py-12 text-base-content/50">
          <svg class="h-12 w-12 mx-auto mb-4 opacity-50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <p>暂无可用版本</p>
          <p class="text-sm mt-2">请检查网络连接</p>
        </div>

        <div v-else class="overflow-auto max-h-[60vh]">
          <table class="table">
            <thead class="sticky top-0 bg-base-100">
              <tr>
                <th>版本</th>
                <th>发布日期</th>
                <th>大小</th>
                <th>状态</th>
                <th class="text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="version in versionStore.remoteVersions" :key="version.version">
                <td class="font-medium">v{{ version.version }}</td>
                <td class="text-base-content/60">{{ formatDate(version.releaseDate) }}</td>
                <td class="text-base-content/60">{{ formatSize(version.size) }}</td>
                <td>
                  <span v-if="version.isLocal" class="badge badge-ghost badge-sm">已安装</span>
                  <span v-else class="badge badge-outline badge-sm">未安装</span>
                </td>
                <td class="text-right">
                  <button
                    v-if="!version.isLocal && versionStore.downloading !== version.version"
                    class="btn btn-primary btn-sm"
                    @click="handleDownload(version.version)"
                  >
                    下载
                  </button>
                  <div v-else-if="versionStore.downloading === version.version" class="flex flex-col items-end gap-1 min-w-32">
                    <progress class="progress progress-primary w-24" :value="versionStore.downloadProgress?.percent || 0" max="100"></progress>
                    <span class="text-xs text-base-content/60">
                      {{ versionStore.downloadProgress?.percent?.toFixed(1) || 0 }}%
                      <span v-if="versionStore.downloadProgress?.speed"> - {{ formatSpeed(versionStore.downloadProgress.speed) }}</span>
                    </span>
                  </div>
                  <span v-else class="text-success text-sm">✓</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="modal-action">
          <button class="btn" @click="showRemoteDialog = false">关闭</button>
        </div>
      </div>
      <div class="modal-backdrop" @click="showRemoteDialog = false"></div>
    </dialog>

    <!-- 删除确认对话框 -->
    <dialog :class="['modal', { 'modal-open': showDeleteDialog }]">
      <div class="modal-box">
        <h3 class="font-bold text-lg">确认删除</h3>
        <p class="py-4">
          确定要删除版本 <span class="font-bold">v{{ deleteTargetVersion }}</span> 吗？
        </p>
        <div class="modal-action">
          <button class="btn btn-ghost" @click="showDeleteDialog = false">取消</button>
          <button class="btn btn-error" @click="handleDeleteConfirm">删除</button>
        </div>
      </div>
      <div class="modal-backdrop" @click="showDeleteDialog = false"></div>
    </dialog>
  </div>
</template>
