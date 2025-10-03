import axios from 'axios';
import { ElMessage, ElMessageBox } from 'element-plus';
import store from '@/store';
import { getToken } from '@/utils/auth';

const AUTH_BASE_URL = import.meta.env.VITE_AUTH_API || 'https://api.yangshujie.com/v1';
const ADMIN_BASE_URL = import.meta.env.VITE_ADMIN_API || 'https://api.yangshujie.com/v1/admin';

const service = axios.create({
  timeout: 5000,
  transformResponse: [(data) => {
    if (!data || typeof data !== 'string') {
      return data;
    }
    const processed = data.replace(/"id":(\d{15,})/g, '"id":"$1"');
    try {
      return JSON.parse(processed);
    } catch (error) {
      return processed;
    }
  }]
});

service.interceptors.request.use(
  config => {
    if (config.url && config.url.startsWith('/auth/')) {
      config.baseURL = AUTH_BASE_URL;
    } else {
      config.baseURL = ADMIN_BASE_URL;
    }

    const token = getToken();
    if (token) {
      config.headers = config.headers || {};
      config.headers.Authorization = `Bearer ${token}`;
    }

    return config;
  },
  error => {
    console.error(error);
    return Promise.reject(error);
  }
);

service.interceptors.response.use(
  response => {
    const res = response.data || {};

    if (res.code !== 'ok') {
      ElMessage({
        message: res.message || '请求失败',
        type: 'error',
        duration: 5 * 1000
      });

      if (res.code === 'unauthorized') {
        ElMessageBox.confirm('登录状态失效，请重新登录', '提示', {
          confirmButtonText: '重新登录',
          cancelButtonText: '取消',
          type: 'warning'
        }).then(() => {
          store.user().resetToken();
          window.location.reload();
        });
      }

      return Promise.reject(new Error(res.message || '请求失败'));
    }

    return res.payload;
  },
  error => {
    console.error('request error:', error);
    ElMessage({
      message: error.message,
      type: 'error',
      duration: 5 * 1000
    });
    return Promise.reject(error);
  }
);

export default service;
