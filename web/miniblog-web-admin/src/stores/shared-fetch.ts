const requests = new WeakMap<object, Map<string, Promise<void>>>();
export function shareFetch(owner: object, key: string, load: () => Promise<void>): Promise<void> {
  let map = requests.get(owner);
  if (!map) { map = new Map(); requests.set(owner, map); }
  const existing = map.get(key);
  if (existing) return existing;
  const pending = Promise.resolve().then(load).finally(() => { if (map?.get(key) === pending) map.delete(key); });
  map.set(key, pending);
  return pending;
}
export function bySort<T extends { sort?: number; code: string }>(items: T[]): T[] {
  return [...items].sort((a, b) => (a.sort || 0) - (b.sort || 0) || a.code.localeCompare(b.code));
}
