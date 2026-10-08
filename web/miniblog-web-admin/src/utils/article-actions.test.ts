import { describe, expect, it } from 'vitest';
import { canArticleAction, sourceURL } from './article-actions';
import type { ArticleInfo } from '@/types/content';
const article = { id: '9007199254740993', status: 'Published', external_link: 'https://example.com/raw' } as ArticleInfo;
describe('article operation capabilities', () => {
  it('fails closed for managed records lacking actions and honors the backend whitelist', () => {
    const managed = { ...article, management: { mode: 'notion_sync' as const, managed_fields: ['title'] }};
    expect(canArticleAction(managed, 'edit')).toBe(false); expect(canArticleAction(managed, 'move')).toBe(false);
    expect(canArticleAction({ ...managed, allowed_actions: ['edit_local_fields', 'reorder', 'hold'] }, 'reorder')).toBe(true);
    expect(canArticleAction({ ...managed, allowed_actions: [] }, 'hold')).toBe(false);
    expect(canArticleAction({ ...managed, allowed_actions: ['hold'] }, 'unpublish')).toBe(false);
  });
  it('keeps old manual servers usable and validates both source URL candidates', () => {
    expect(canArticleAction(article, 'unpublish')).toBe(true); expect(canArticleAction(article, 'move')).toBe(true);
    expect(sourceURL({ ...article, reading_url: 'javascript:bad' })).toBe(article.external_link);
    expect(sourceURL({ ...article, reading_url: 'https://example.com/current' })).toBe('https://example.com/current');
  });
});
