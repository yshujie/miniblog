import request from '@/utils/request';

export function fetchAllSections() {
  return request({
    url: '/sections',
    method: 'get'
  });
}

export function fetchSections(moduleCode) {
  return request({
    url: `/sections/${moduleCode}`,
    method: 'get'
  });
}
