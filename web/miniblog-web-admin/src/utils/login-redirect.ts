import type { LocationQuery } from 'vue-router';
export function localLoginTarget(value: unknown) {
  return typeof value === 'string' && value.startsWith('/') && !value.startsWith('//') && !value.includes('\\') ? value : '/home';
}
export function loginDestination(target: unknown, extraQuery: LocationQuery = {}) {
  const url = new URL(localLoginTarget(target), 'http://miniblog.local');
  const query: LocationQuery = {};
  url.searchParams.forEach((value, key) => { const current = query[key]; query[key] = current == null ? value : Array.isArray(current) ? [...current, value] : [current, value]; });
  return { path: url.pathname, query: { ...query, ...extraQuery }, hash: url.hash };
}
