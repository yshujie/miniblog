// 通用响应类型
export interface ApiResponse<T> {
  code: string
  message?: string
  msg?: string
  payload: T
}
