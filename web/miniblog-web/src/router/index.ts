import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

const routes: Array<RouteRecordRaw> = [
  { path: '/', component: () => import('../pages/Index.vue') },
  { path: '/blog/:module', component: () => import('../pages/Blog.vue') },
  { path: '/blog/:module/:article', component: () => import('../pages/Blog.vue') },
  { path: '/404', component: () => import('../pages/NotPage.vue') },
  { path: '/:pathMatch(.*)*', redirect: '/404' },
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
