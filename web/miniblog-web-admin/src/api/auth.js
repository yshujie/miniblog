import request from '@/utils/request'

/**
 * 登录
 * @param {Object} data 登录数据
 * @returns {Promise} 登录结果
 */
export function login(data) {
  return request({
    url: '/auth/login',
    method: 'post',
    data
  })
}

/**
 * 退出登录
 * @returns {Promise} 退出登录结果
 */
export function logout() {
  return request({
    url: '/auth/logout',
    method: 'post'
  })
}
