import http from '../util/http'

export interface LoginResponse {
  token: string
  user: {
    id: number
    username: string
  }
}

export async function login(username: string, password: string): Promise<LoginResponse> {
  const { data } = await http.post<LoginResponse>('/api/v1/login', {
    username,
    password
  })
  // 保存 token 到 localStorage
  localStorage.setItem('token', data.token)
  return data
}

export interface RegisterResponse {
  id: number
  username: string
}

export async function register(username: string, password: string): Promise<RegisterResponse> {
  const { data } = await http.post<RegisterResponse>('/api/v1/register', {
    username,
    password
  })
  return data
}

export async function logout(): Promise<void> {
  localStorage.removeItem('token')
  await http.post('/api/v1/logout')
}
