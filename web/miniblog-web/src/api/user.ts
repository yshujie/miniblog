export async function login(username: string, password: string): Promise<boolean> {
  // mock 验证
  return new Promise(resolve =>
    setTimeout(() => resolve(username === 'admin' && password === 'admin'), 300)
  )
}

export async function register(username: string, password: string): Promise<boolean> {
  // mock 注册
  return new Promise(resolve =>
    setTimeout(() => resolve(username !== 'admin'), 300)
  )
}
