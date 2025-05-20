import http from '../util/http'

export interface LoginResponse {
  token: string
  user: {
    id: number
    username: string
  }
}

export async function login(username: string, password: string): Promise<LoginResponse> {
  const { payload } = await http.post<LoginResponse>('/api/v1/login', {
    username,
    password
  })
  // 保存 token 到 localStorage
  localStorage.setItem('token', payload.token)
  return payload
}

export interface RegisterResponse {
  id: number
  username: string
}

export async function register(username: string, password: string): Promise<RegisterResponse> {
  const { payload } = await http.post<RegisterResponse>('/api/v1/register', {
    username,
    password
  })
  return payload
}

export async function logout(): Promise<void> {
  localStorage.removeItem('token')
  await http.post('/api/v1/logout')
}
