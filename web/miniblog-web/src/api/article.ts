import http from '@/util/http'
import { Article } from '@/types/article'

// fetchArticlesBySectionCode 获取分类文章列表
export async function fetchArticlesBySectionCode(sectionCode: string): Promise<Article[]> {
  const { payload } = await http.get<{ articles: any[] }>(`/articles/${sectionCode}`)
  return payload.articles.map(articleData => new Article(articleData))
}

// fetchArticleById 获取文章详情
export async function fetchArticleById(sectionCode: string, id: number): Promise<Article | undefined> {
  const { payload } = await http.get<{ article: any }>(`/articles/${sectionCode}/${id}`)
  return payload.article ? new Article(payload.article) : undefined
}
