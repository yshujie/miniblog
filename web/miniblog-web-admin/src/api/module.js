import request from '@/utils/request';

export function fetchModules() {
  return request({
    url: '/modules',
    method: 'get'
  });
}
