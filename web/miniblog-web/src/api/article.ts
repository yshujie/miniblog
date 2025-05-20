import http from '@/util/http'
import type { Article } from '../types/article'

// fetchArticlesBySectionCode 获取分类文章列表
export async function fetchArticlesBySectionCode(sectionCode: string): Promise<Article[]> {
  const { payload } = await http.get<{ articles: Article[] }>(`/articles?section_code=${sectionCode}`)
  return payload.articles
}

// fetchArticleById 获取文章详情
export async function fetchArticleById(id: number): Promise<Article | undefined> {
  const { payload } = await http.get<{ article: Article }>(`/articles/${id}`)
  return payload.article
}
