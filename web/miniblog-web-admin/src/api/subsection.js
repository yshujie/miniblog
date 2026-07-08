import request from '@/utils/request';

export function fetchSubsections(sectionCode) {
  return request({
    url: `/subsections/${sectionCode}`,
    method: 'get'
  });
}

export function createSubsection(data) {
  return request({
    url: '/subsections',
    method: 'post',
    data
  });
}

export function updateSubsection(code, data) {
  return request({
    url: `/subsections/${code}`,
    method: 'put',
    data
  });
}

export function publishSubsection(code) {
  return request({
    url: `/subsections/${code}/publish`,
    method: 'put'
  });
}

export function unpublishSubsection(code) {
  return request({
    url: `/subsections/${code}/unpublish`,
    method: 'put'
  });
}

export function deleteSubsection(code) {
  return request({
    url: `/subsections/${code}`,
    method: 'delete'
  });
}
