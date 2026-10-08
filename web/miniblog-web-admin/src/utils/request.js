import axios from 'axios';
import { ElMessage, ElMessageBox } from 'element-plus';
import store from '@/store';
import { getToken } from '@/utils/auth';
import { ApiError } from '@/utils/api-error';

const apiRoot = (import.meta.env.VITE_API_ROOT || 'http://localhost:8080/v1').replace(/\/$/, '');
const AUTH_BASE_URL = apiRoot;
const ADMIN_BASE_URL = `${apiRoot}/admin`;

// Content pages own their inline errors; legacy login and user screens keep notifications.
const isContentRequest = config => /^\/(articles|article-sources|catalog|modules|sections|subsections|notion-sync)(\/|$)/.test(config?.url || '');

const service = axios.create({
  timeout: 5000,
  transformResponse: [(data) => {
    if (!data || typeof data !== 'string') {
      return data;
    }
    const processed = data.replace(/"id":(\d{15,})/g, '"id":"$1"');
    try {
      return JSON.parse(processed);
    } catch {
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
      if (!isContentRequest(response.config)) ElMessage.error(res.message || '请求失败');
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

      return Promise.reject(new ApiError(res.message || '请求失败', res.code, response.status));
    }

    return res.payload;
  },
  error => {
    const data = error.response?.data || {};
    if (!isContentRequest(error.config)) ElMessage.error(data.message || error.message || '网络请求失败');
    return Promise.reject(new ApiError(data.message || error.message || '网络请求失败', data.code || error.code, error.response?.status || 0));
  }
);

export default service;
