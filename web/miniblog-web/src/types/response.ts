// 通用响应类型
export interface ApiResponse<T> {
  code: string
  msg: string
  payload: T
} 