import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import './assets/base.css'

// 定义 app
const app = createApp(App)

// 使用插件
const pinia = createPinia()
app.use(pinia) // 使用 pinia
app.use(router) // 使用路由
app.use(ElementPlus) // 使用 element-plus

app.mount('#app')
