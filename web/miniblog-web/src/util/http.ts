import axios from 'axios'
import type { ApiResponse } from '../types/response'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api/v1'

// 创建 axios 实例
const http = axios.create({
  baseURL: API_BASE_URL, // dev 默认走 Vite 代理
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
    // 直接返回响应数据，因为后端已经包装了 code、msg 和 payload
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

// 扩展 axios 实例的类型
declare module 'axios' {
  interface AxiosInstance {
    get<T = any>(url: string, config?: any): Promise<ApiResponse<T>>
    post<T = any>(url: string, data?: any, config?: any): Promise<ApiResponse<T>>
    put<T = any>(url: string, data?: any, config?: any): Promise<ApiResponse<T>>
    delete<T = any>(url: string, config?: any): Promise<ApiResponse<T>>
    patch<T = any>(url: string, data?: any, config?: any): Promise<ApiResponse<T>>
  }
}

export default http 
