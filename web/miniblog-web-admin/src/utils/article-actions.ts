import type { ArticleInfo } from '@/types/content';
export const isManagedArticle = (article?: ArticleInfo) => article?.management?.mode === 'notion_sync';
export function canArticleAction(article: ArticleInfo | undefined, action: string): boolean {
  if (!article) return false;
  if (article.allowed_actions) return article.allowed_actions.includes(action);
  // Older servers have no ownership contract. Managed rows fail closed.
  if (isManagedArticle(article)) return action === 'view_source';
  if (action === 'view_source') return true;
  if (['edit', 'move', 'archive', 'reorder'].includes(action)) return article.status !== 'Deleted';
  if (action === 'publish') return !['Deleted', 'Published'].includes(article.status);
  if (action === 'unpublish') return article.status === 'Published';
  if (action === 'restore') return article.status === 'Deleted';
  return false;
}
export function sourceURL(article: ArticleInfo): string {
  for (const value of [article.reading_url, article.external_link]) {
    try { const url = new URL(value || ''); if (['http:', 'https:'].includes(url.protocol)) return url.href; } catch { /* Try the captured original URL. */ }
  }
  return '';
}
