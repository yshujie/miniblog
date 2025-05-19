import axios from 'axios'

// 创建 axios 实例
const http = axios.create({
  baseURL: 'https://api.yangshujie.com/v1', // 修改为您的实际服务器地址
  timeout: 5000, // 请求超时时间
  headers: {
    'Content-Type': 'application/json'
  }
})

// 设置新的 API 地址
export function setBaseURL(url: string) {
  http.defaults.baseURL = url
}

// 获取当前 API 地址
export function getBaseURL(): string {
  return http.defaults.baseURL || ''
}

// 请求拦截器
http.interceptors.request.use(
  config => {
    // 从 localStorage 获取 token
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  error => {
    return Promise.reject(error)
  }
)

// 响应拦截器
http.interceptors.response.use(
  response => {
    return response.data
  },
  error => {
    if (error.response) {
      switch (error.response.status) {
        case 401:
          // 未授权，清除 token 并跳转到登录页
          localStorage.removeItem('token')
          window.location.href = '/login'
          break
        case 403:
          // 权限不足
          console.error('没有权限访问该资源')
          break
        case 404:
          // 资源不存在
          console.error('请求的资源不存在')
          break
        default:
          console.error('服务器错误')
      }
    }
    return Promise.reject(error)
  }
)

export default http 