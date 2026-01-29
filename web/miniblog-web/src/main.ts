import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import { useModuleStore } from './stores/module'
import './assets/base.css'

// 定义 app
const app = createApp(App)

// 使用插件
app.use(router) // 使用路由
const pinia = createPinia()
app.use(pinia) // 使用 pinia
app.use(ElementPlus) // 使用 element-plus

// 预加载模块数据
async function initApp() {
  try {
    const moduleStore = useModuleStore()
    await moduleStore.loadModules()
    console.log('✅ 模块数据预加载成功')
  } catch (error) {
    console.error('❌ 模块数据预加载失败:', error)
  }

  // 挂载到 app
  app.mount('#app')
}

// 启动应用
initApp()
