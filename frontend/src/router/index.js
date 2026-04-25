import { createRouter, createWebHistory } from 'vue-router'
import ConfigsView from '../views/ConfigsView.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      component: ConfigsView
    },
    {
      path: '/configs',
      name: 'configs',
      component: ConfigsView
    },
    {
      path: '/status',
      name: 'status',
      component: () => import('../views/StatusView.vue')
    },
    {
      path: '/version',
      name: 'version',
      component: () => import('../views/VersionView.vue')
    },
    {
      path: '/logs',
      name: 'logs',
      component: () => import('../views/LogsView.vue')
    },
    {
      path: '/settings',
      name: 'settings',
      component: () => import('../views/SettingsView.vue')
    }
  ]
})

export default router
