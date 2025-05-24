import defaultSettings from '@/settings'

/**
 * 页面标题
 */
const title = defaultSettings.title || 'Vue Element Admin'

/**
 * 获取页面标题
 * @param {String} pageTitle 页面标题
 * @returns {String} 页面标题
 */
export default function getPageTitle(pageTitle) {
  if (pageTitle) {
    return `${pageTitle} - ${title}`
  }
  return `${title}`
}
