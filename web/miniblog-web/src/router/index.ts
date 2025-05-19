import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

const routes: Array<RouteRecordRaw> = [
  { path: '/', component: () => import('../pages/Index.vue') },
  { path: '/blog/:module', component: () => import('../pages/Blog.vue') },
  { path: '/blog/:module/:article', component: () => import('../pages/Blog.vue') },
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
