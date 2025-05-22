import request from '@/utils/request'

export function login(data) {
  return request({
    url: '/auth/login',
    method: 'post',
    data
  })
}

export function logout() {
  return request({
    url: '/auth/logout',
    method: 'post'
  })
}

export function getInfo(token) {
  console.log('in getInfo', token)
  return request({
    url: '/users/myinfo',
    method: 'get'
  })
}
