import request from '@/utils/request'

/**
 * 获取文章列表
 * @param {Object} query 查询参数
 * @returns {Promise} 文章列表
 */
export function fetchList(query) {
  return request({
    url: '/articles',
    method: 'get',
    params: query
  })
}

/**
 * 获取文章详情
 * @param {String} id 文章 ID
 * @returns {Promise} 文章详情
 */
export function fetchArticle(id) {
  return request({
    url: '/article/detail',
    method: 'get',
    params: { id }
  })
}

/**
 * 创建文章
 * @param {Object} data 文章数据
 * @returns {Promise} 创建结果
 */
export function createArticle(data) {
  return request({
    url: '/article/create',
    method: 'post',
    data
  })
}

/**
 * 更新文章
 * @param {Object} data 文章数据
 * @returns {Promise} 更新结果
 */
export function updateArticle(data) {
  return request({
    url: '/article/update',
    method: 'post',
    data
  })
}
