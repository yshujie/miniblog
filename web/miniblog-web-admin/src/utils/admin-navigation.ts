import type { RouteRecordRaw } from 'vue-router';
interface NavigationItem { name: string; path: string; title: string; icon: string }
export function adminNavigation(routes: RouteRecordRaw[]) {
  const available = new Set<string>();
  function visit(items: RouteRecordRaw[]) { items.forEach(item => { if (item.name) available.add(String(item.name)); if (item.children) visit(item.children); }); }
  visit(routes);
  const groups: Array<{ title: string; items: NavigationItem[] }> = [
    { title: '', items: [{ name: 'Home', path: '/home', title: '工作概览', icon: 'dashboard' }] },
    { title: '内容工作台', items: [{ name: 'ContentSync', path: '/content/sync', title: 'Notion 自动同步', icon: 'tree' }, { name: 'ContentWorkbench', path: '/content/workbench', title: '目录与收录', icon: 'list' }] },
    { title: '内容管理', items: [{ name: 'ArticleList', path: '/article/list', title: '文章库', icon: 'documentation' }] }
  ];
  return groups.map(group => ({ ...group, items: group.items.filter(item => available.has(item.name)) })).filter(group => group.items.length);
}
