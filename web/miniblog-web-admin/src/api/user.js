import request from '@/utils/request'

/**
 * 获取用户信息
 * @param {String} token 令牌
 * @returns {Promise} 用户信息
 */
export function getInfo(token) {
  console.log('in getInfo', token)
  return request({
    url: '/users/myinfo',
    method: 'get'
  })
}
