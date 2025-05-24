import request from '@/utils/request'

/**
 * 搜索用户
 * @param {String} name 用户名
 * @returns {Promise} 用户列表
 */
export function searchUser(name) {
  return request({
    url: '/search/user',
    method: 'get',
    params: { name }
  })
}

/**
 * 获取交易列表
 * @param {Object} query 查询参数
 * @returns {Promise} 交易列表
 */
export function transactionList(query) {
  return request({
    url: '/transaction/list',
    method: 'get',
    params: query
  })
}
