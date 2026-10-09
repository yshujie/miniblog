import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

export const routes: RouteRecordRaw[] = [
  { path: '/', component: () => import('../pages/Index.vue') },
  { path: '/blog/:module', component: () => import('../pages/Blog.vue'), name: 'BlogModule', meta: { layout: 'reading' } },
  { path: '/blog/:module/article/:article', component: () => import('../pages/Blog.vue'), name: 'BlogArticle', meta: { layout: 'reading' } },
  { path: '/topics/:module', component: () => import('../pages/TopicOverview.vue'), name: 'TopicOverview' },
  { path: '/404', component: () => import('../pages/NotPage.vue') },
  { path: '/:pathMatch(.*)*', redirect: '/404' },
]

// The page resolves Article ID before checking a historical module parameter.
export default createRouter({
  history: createWebHistory(), routes,
  scrollBehavior(to, _from, savedPosition) {
    if (savedPosition) return savedPosition
    // TopicOverview locates the chapter after its shared directory request settles.
    if (to.name === 'TopicOverview' && typeof to.query.chapter === 'string') return false
    if (to.path === '/' && to.hash === '#topics') return { el: '#topics' }
    return { left: 0, top: 0 }
  },
})
