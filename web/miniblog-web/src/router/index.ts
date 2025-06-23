import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useModuleStore } from '@/stores/module'

const routes: Array<RouteRecordRaw> = [
  { path: '/', component: () => import('../pages/Index.vue') },
  {
    path: '/blog/:module',
    component: () => import('../pages/Blog.vue'),
    name: 'BlogModule'
  },
  {
    path: '/blog/:module/article/:article',
    component: () => import('../pages/Blog.vue'),
    name: 'BlogArticle'
  },
  { path: '/404', component: () => import('../pages/NotPage.vue') },
  { path: '/:pathMatch(.*)*', redirect: '/404' },
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// 路由守卫：确保访问博客页面时模块数据已加载
router.beforeEach(async (to, from, next) => {
  // 如果访问的是博客页面，确保模块数据已加载
  if (to.name === 'BlogModule' || to.name === 'BlogArticle') {
    const moduleStore = useModuleStore()

    // 如果模块数据还没加载，先加载
    if (moduleStore.modules.length === 0) {
      try {
        console.log('🔄 路由守卫：预加载模块数据...')
        await moduleStore.loadModules()
      } catch (error) {
        console.error('❌ 路由守卫：模块数据加载失败', error)
        next('/404')
        return
      }
    }
  }

  next()
})

export default router
