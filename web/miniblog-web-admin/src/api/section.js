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

export function updateSection(code, data) {
  return request({
    url: `/sections/${code}`,
    method: 'put',
    data
  });
}

export function publishSection(code) {
  return request({
    url: `/sections/${code}/publish`,
    method: 'put'
  });
}

export function unpublishSection(code) {
  return request({
    url: `/sections/${code}/unpublish`,
    method: 'put'
  });
}

export function createSection(data) {
  return request({
    url: '/sections',
    method: 'post',
    data
  });
}

export function deleteSection(code) {
  return request({
    url: `/sections/${code}`,
    method: 'delete'
  });
}
