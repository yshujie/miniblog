import request from '@/utils/request';

export function fetchModules() {
  return request({
    url: '/modules',
    method: 'get'
  });
}

export function createModule(data) {
  return request({
    url: '/modules',
    method: 'post',
    data
  });
}
