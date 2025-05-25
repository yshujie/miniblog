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
    url: '/articles/' + id,
    method: 'get'
  })
}

/**
 * 创建文章
 * @param {Object} data 文章数据
 * @returns {Promise} 创建结果
 */
export function createArticle(data) {
  return request({
    url: '/articles',
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
    url: '/articles/' + data.id,
    method: 'put',
    data
  })
}

/**
 * 发布文章
 * @param {Object} data 文章数据
 * @returns {Promise} 发布结果
 */
export function publishArticle(data) {
  return request({
    url: '/article/publish',
    method: 'post',
    data
  })
}
