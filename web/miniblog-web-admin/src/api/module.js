import request from '@/utils/request'

/**
 * 获取所有模块
 * @returns {Promise} 模块列表
 */
export function fetchModules() {
  return request({ url: '/modules', method: 'get' })
}
