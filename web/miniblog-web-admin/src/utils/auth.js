const TokenKey = 'Admin-Token'

// 获取 token
export function getToken() {
  return localStorage.getItem(TokenKey)
}

// 设置 token
export function setToken(token) {
  console.log('in setToken', token)

  // local storage
  localStorage.setItem(TokenKey, token)

  return localStorage.setItem(TokenKey, token)
}

// 删除 token
export function removeToken() {
  return localStorage.removeItem(TokenKey)
}
