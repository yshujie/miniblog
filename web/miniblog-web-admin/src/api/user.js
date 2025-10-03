import request from '@/utils/request';

export function getInfo() {
  return request({
    url: '/users/myinfo',
    method: 'get'
  });
}
