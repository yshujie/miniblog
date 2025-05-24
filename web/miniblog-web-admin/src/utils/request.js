import axios from 'axios'
import { MessageBox, Message } from 'element-ui'
import store from '@/store'
import { getToken } from '@/utils/auth'

// 定义基础 URL
const AUTH_BASE_URL = 'https://api.yangshujie.com/api/auth'
const ADMIN_BASE_URL = 'https://api.yangshujie.com/api/admin'

/**
 * 创建 axios 实例
 * @returns {Object} axios 实例
 */
const service = axios.create({
  timeout: 5000 // 请求超时时间
})

/**
 * 请求拦截器
 * @param {Object} config 请求配置
 * @returns {Promise} 请求配置
 */
service.interceptors.request.use(
  config => {
    // 根据请求路径设置不同的 baseURL
    if (config.url.startsWith('/auth/')) {
      config.baseURL = AUTH_BASE_URL
    } else {
      config.baseURL = ADMIN_BASE_URL
    }

    // 如果存在 token，则添加到请求头
    if (getToken()) {
      // 添加 token 到请求头
      config.headers['Authorization'] = `Bearer ${getToken()}`
    }
    return config
  },
  error => {
    // 处理请求错误
    console.log(error) // 调试
    return Promise.reject(error)
  }
)

/**
 * 响应拦截器
 * @param {Object} response 响应数据
 * @returns {Promise} 响应数据
 */
service.interceptors.response.use(
  /**
   * Determine the request status by custom code
   * Here is just an example
   * You can also judge the status by HTTP Status Code
   */
  response => {
    const res = response.data

    // 如果响应码不是 ok，则提示错误
    if (res.code !== 'ok') {
      Message({
        message: res.message || 'Error',
        type: 'error',
        duration: 5 * 1000
      })

      // 如果响应码是 unauthorized，则提示重新登录
      if (res.code === 'unauthorized') {
        // to re-login
        MessageBox.confirm('You have been logged out, you can cancel to stay on this page, or log in again', 'Confirm logout', {
          confirmButtonText: 'Re-Login',
          cancelButtonText: 'Cancel',
          type: 'warning'
        }).then(() => {
          store.dispatch('user/resetToken').then(() => {
            location.reload()
          })
        })
      }

      // 返回错误信息
      return Promise.reject(new Error(res.message || 'Error'))
    } else {
      return res
    }
  },
  error => {
    console.log('err' + error) // for debug
    Message({
      message: error.message,
      type: 'error',
      duration: 5 * 1000
    })
    return Promise.reject(error)
  }
)

export default service
