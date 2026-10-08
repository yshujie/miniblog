import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  { path: '/', component: () => import('../pages/Index.vue') },
  { path: '/blog/:module', component: () => import('../pages/Blog.vue'), name: 'BlogModule' },
  { path: '/blog/:module/article/:article', component: () => import('../pages/Blog.vue'), name: 'BlogArticle' },
  { path: '/404', component: () => import('../pages/NotPage.vue') },
  { path: '/:pathMatch(.*)*', redirect: '/404' },
]

// The page resolves Article ID before checking a historical module parameter.
export default createRouter({ history: createWebHistory(), routes })
