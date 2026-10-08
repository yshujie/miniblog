import axios, { type AxiosRequestConfig, type Method } from 'axios'
import type { ApiResponse } from '@/types/response'

export class ApiError extends Error {
  constructor(message: string, readonly status: number, readonly code = '') {
    super(message)
    this.name = 'ApiError'
  }
}

// The legacy module list emits numeric IDs. Preserve them before JSON number conversion.
export function parseResponse(data: unknown): unknown {
  if (typeof data !== 'string') return data
  return JSON.parse(data.replace(/("id"\s*:\s*)(\d+)(?=\s*[,}])/g, '$1"$2"'))
}

const client = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? (import.meta.env.PROD && import.meta.env.MODE !== 'test' ? 'https://api.yangshujie.com/v1' : '/api/v1'),
  timeout: 5000,
  headers: { 'Content-Type': 'application/json' },
  transformResponse: [parseResponse],
})

client.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = 'Bearer ' + token
  return config
})

async function request<T>(method: Method, url: string, data?: unknown, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
  try {
    const response = await client.request<ApiResponse<T>>({ ...config, method, url, data })
    if (response.data.code !== 'ok') {
      throw new ApiError(response.data.message || response.data.msg || '请求失败，请重试', response.status, response.data.code)
    }
    return response.data
  } catch (error) {
    if (axios.isAxiosError(error) && error.code !== 'ERR_CANCELED') {
      const body = error.response?.data as Partial<ApiResponse<unknown>> | undefined
      throw new ApiError(body?.message || body?.msg || '请求失败，请检查网络后重试', error.response?.status ?? 0, body?.code)
    }
    throw error
  }
}

export const setBaseURL = (url: string) => { client.defaults.baseURL = url }
export const getBaseURL = () => client.defaults.baseURL || ''

export default {
  get: <T>(url: string, config?: AxiosRequestConfig) => request<T>('GET', url, undefined, config),
  post: <T>(url: string, data?: unknown, config?: AxiosRequestConfig) => request<T>('POST', url, data, config),
  put: <T>(url: string, data?: unknown, config?: AxiosRequestConfig) => request<T>('PUT', url, data, config),
  patch: <T>(url: string, data?: unknown, config?: AxiosRequestConfig) => request<T>('PATCH', url, data, config),
  delete: <T>(url: string, config?: AxiosRequestConfig) => request<T>('DELETE', url, undefined, config),
}
