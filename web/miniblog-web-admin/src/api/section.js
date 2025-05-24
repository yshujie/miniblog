import request from '@/utils/request'

/**
 * 获取所有章节
 * @returns {Promise} 章节列表
 */
export function fetchAllSections() {
  return request({
    url: '/sections',
    method: 'get'
  })
}

/**
 * 获取指定模块的章节
 * @param {String} moduleCode 模块代码
 * @returns {Promise} 章节列表
 */
export function fetchSections(moduleCode) {
  return request({
    url: `/sections/${moduleCode}`,
    method: 'get'
  })
}
