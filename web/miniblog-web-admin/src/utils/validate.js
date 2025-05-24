/**
 * Created by PanJiaChen on 16/11/18.
 */

/**
 * 检查是否为外部链接
 * @param {String} path 路径
 * @returns {Boolean} 是否为外部链接
 */
export function isExternal(path) {
  return /^(https?:|mailto:|tel:)/.test(path)
}

/**
 * 检查是否为有效用户名
 * @param {String} str 用户名
 * @returns {Boolean} 是否为有效用户名
 */
export function validUsername(str) {
  return true
}

/**
 * 检查是否为有效 URL
 * @param {String} url URL
 * @returns {Boolean} 是否为有效 URL
 */
export function validURL(url) {
  const reg = /^(https?|ftp):\/\/([a-zA-Z0-9.-]+(:[a-zA-Z0-9.&%$-]+)*@)*((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])){3}|([a-zA-Z0-9-]+\.)*[a-zA-Z0-9-]+\.(com|edu|gov|int|mil|net|org|biz|arpa|info|name|pro|aero|coop|museum|[a-zA-Z]{2}))(:[0-9]+)*(\/($|[a-zA-Z0-9.,?'\\+&%$#=~_-]+))*$/
  return reg.test(url)
}

/**
 * 检查是否为有效小写字母
 * @param {String} str 字符串
 * @returns {Boolean} 是否为有效小写字母
 */
export function validLowerCase(str) {
  const reg = /^[a-z]+$/
  return reg.test(str)
}

/**
 * 检查是否为有效大写字母
 * @param {String} str 字符串
 * @returns {Boolean} 是否为有效大写字母
 */
export function validUpperCase(str) {
  const reg = /^[A-Z]+$/
  return reg.test(str)
}

/**
 * 检查是否为有效字母
 * @param {String} str 字符串
 * @returns {Boolean} 是否为有效字母
 */
export function validAlphabets(str) {
  const reg = /^[A-Za-z]+$/
  return reg.test(str)
}

/**
 * 检查是否为有效邮箱
 * @param {String} email 邮箱
 * @returns {Boolean} 是否为有效邮箱
 */
export function validEmail(email) {
  const reg = /^(([^<>()\[\]\\.,;:\s@"]+(\.[^<>()\[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/
  return reg.test(email)
}

/**
 * 检查是否为字符串
 * @param {String} str 字符串
 * @returns {Boolean} 是否为字符串
 */
export function isString(str) {
  if (typeof str === 'string' || str instanceof String) {
    return true
  }
  return false
}

/**
 * 检查是否为数组
 * @param {Array} arg 数组
 * @returns {Boolean} 是否为数组
 */
export function isArray(arg) {
  if (typeof Array.isArray === 'undefined') {
    return Object.prototype.toString.call(arg) === '[object Array]'
  }
  return Array.isArray(arg)
}
