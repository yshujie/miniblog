import { createRouter, createWebHistory, RouteRecordRaw } from 'vue-router'

const routes: Array<RouteRecordRaw> = [
  { path: '/', component: () => import('../pages/Home.vue') },
  { path: '/article/:id', component: () => import('../pages/ArticleDetail.vue') },
  { path: '/archive', component: () => import('../pages/Archive.vue') },
  { path: '/tag/:name', component: () => import('../pages/Tag.vue') },
  { path: '/friend', component: () => import('../pages/Friend.vue') },
  { path: '/about', component: () => import('../pages/About.vue') },
  { path: '/login', component: () => import('../pages/Login.vue') },
  { path: '/register', component: () => import('../pages/Register.vue') }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
