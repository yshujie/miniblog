import { describe, expect, it } from 'vitest';
import { loginDestination } from './login-redirect';
describe('login return destination', () => {
  it('retains exact large IDs, collection context and hash', () => {
    expect(loginDestination('/article/edit/9007199254740993?module_code=go&collect=1#source')).toEqual({ path: '/article/edit/9007199254740993', query: { module_code: 'go', collect: '1' }, hash: '#source' });
  });
  it('rejects external, protocol relative and backslash targets', () => {
    for (const value of ['https://example.invalid', '//example.invalid', '/\\example.invalid', undefined]) expect(loginDestination(value).path).toBe('/home');
  });
  it('retains repeated parameters and explicit additional parameters', () => {
    expect(loginDestination('/content/sync?tab=pages&tag=a&tag=b', { source_id: 'go' }).query).toEqual({ tab: 'pages', tag: ['a', 'b'], source_id: 'go' });
  });
});
