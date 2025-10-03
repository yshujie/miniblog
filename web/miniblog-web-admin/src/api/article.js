import request from '@/utils/request';

export function fetchList(query) {
  return request({
    url: '/articles',
    method: 'get',
    params: query
  });
}

export function fetchArticle(id) {
  return request({
    url: `/articles/${id}`,
    method: 'get'
  });
}

export function createArticle(data) {
  return request({
    url: '/articles',
    method: 'post',
    data
  });
}

export function updateArticle(data) {
  return request({
    url: `/articles/${data.id}`,
    method: 'put',
    data
  });
}

export function publishArticle(data) {
  return request({
    url: `/articles/${data.id}/publish`,
    method: 'put'
  });
}

export function unpublishArticle(data) {
  return request({
    url: `/articles/${data.id}/unpublish`,
    method: 'put'
  });
}
