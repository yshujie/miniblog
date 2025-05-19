import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'

// 定义 app
const app = createApp(App)

// 使用插件
app.use(router) // 使用路由
app.use(createPinia()) // 使用 pinia
app.use(ElementPlus) // 使用 element-plus

// 挂载到 app
app.mount('#app')
