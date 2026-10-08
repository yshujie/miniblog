import request from '@/utils/request';
import type { AxiosRequestConfig } from 'axios';
import type { ArticleInfo, ArticleFilters, ArticleList, CollectRequest, CollectResult, SourcePreview, UpdateArticleRequest, CatalogOrder } from '@/types/content';

type ArticleWire = Omit<ArticleInfo, 'id'> & { id: string | number; id_text?: string };
export function normalizeArticle(article: ArticleWire): ArticleInfo {
  return { ...article, id: article.id_text || String(article.id), tags: article.tags || [], content: article.content || '', author: article.author || '', title: article.title || '' };
}
async function articleResponse(config: AxiosRequestConfig): Promise<{ article: ArticleInfo }> {
  const data = await request(config) as unknown as { article: ArticleWire };
  return { article: normalizeArticle(data.article) };
}
export async function fetchArticles(filters: ArticleFilters): Promise<ArticleList> {
  const data = await request({ url: '/articles', method: 'get', params: filters }) as unknown as { articles: ArticleWire[]; total: number };
  return { articles: (data.articles || []).map(normalizeArticle), total: data.total || 0 };
}
export const getArticle = (id: string) => articleResponse({ url: `/articles/${id}`, method: 'get' });
export const saveArticle = (data: UpdateArticleRequest) => articleResponse({ url: `/articles/${data.id}`, method: 'put', data });
export const createLegacyArticle = (data: UpdateArticleRequest) => articleResponse({ url: '/articles', method: 'post', data });
export async function changeArticleStatus(id: string, action: 'publish' | 'unpublish' | 'archive' | 'restore'): Promise<{ article?: ArticleInfo }> {
  const data = await request({ url: `/articles/${id}/${action}`, method: 'put' }) as unknown as { article?: ArticleWire } | undefined;
  return { article: data?.article ? normalizeArticle(data.article) : undefined };
}
export const moveArticle = (id: string, data: { section_code: string; subsection_code?: string }) => articleResponse({ url: `/articles/${id}/move`, method: 'put', data });
export async function previewSource(external_link: string): Promise<SourcePreview> {
  const data = await request({ url: '/article-sources/preview', method: 'post', data: { external_link }, timeout: 10000 }) as unknown as Omit<SourcePreview, 'existing_article'> & { existing_article?: ArticleWire };
  return { ...data, existing_article: data.existing_article ? normalizeArticle(data.existing_article) : undefined };
}
export async function registerArticle(data: CollectRequest, key: string): Promise<CollectResult> {
  const result = await request({ url: '/articles/register', method: 'post', data, headers: { 'Idempotency-Key': key }, timeout: 10000 }) as unknown as { outcome: CollectResult['outcome']; article: ArticleWire };
  return { outcome: result.outcome, article: normalizeArticle(result.article) };
}
export const reorderArticles = (data: { section_code: string; subsection_code?: string; article_ids: string[] }) => request({ url: '/articles/reorder', method: 'put', data });
export const reorderCatalog = (data: CatalogOrder) => request({ url: '/catalog/reorder', method: 'post', data });
export async function fetchPositionArticles(section_code: string, subsection_code = ''): Promise<ArticleInfo[]> {
  const result: ArticleInfo[] = [];
  let page = 1;
  for (;;) {
    const data = await fetchArticles({ section_code, subsection_code, direct_only: !subsection_code, page, limit: 100 });
    result.push(...data.articles);
    if (result.length >= data.total || !data.articles.length) break;
    page += 1;
  }
  return result.filter(article => article.status !== 'Deleted');
}
