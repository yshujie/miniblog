import { describe, expect, it } from 'vitest';
import { constantRoutes, asyncRoutes } from '@/router';
import { filterAsyncRoutes } from '@/store/modules/permission';
import { adminNavigation } from './admin-navigation';
describe('author navigation and compatible routes', () => {
  it('shows four working entries without removing legacy routes', () => {
    expect(adminNavigation([...constantRoutes, ...asyncRoutes]).flatMap(group => group.items.map(item => item.path))).toEqual(['/home', '/content/sync', '/content/workbench', '/article/list']);
    expect(asyncRoutes.find(item => item.path === '/article')?.children?.map(item => item.path)).toContain('create');
    expect(asyncRoutes.some(item => item.path === '/subsection')).toBe(true);
  });
  it('does not expose protected menus to an unauthorized role', () => {
    expect(adminNavigation([...constantRoutes, ...filterAsyncRoutes(asyncRoutes, ['reader'])]).flatMap(group => group.items.map(item => item.path))).toEqual(['/home']);
  });
});
