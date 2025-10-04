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

export function updateModule(code, data) {
  return request({
    url: `/modules/${code}`,
    method: 'put',
    data
  });
}

export function publishModule(code) {
  return request({
    url: `/modules/${code}/publish`,
    method: 'put'
  });
}

export function unpublishModule(code) {
  return request({
    url: `/modules/${code}/unpublish`,
    method: 'put'
  });
}
