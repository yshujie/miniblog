import http from '@/util/http'
import { Article } from '@/types/article'

// fetchArticlesBySectionCode 获取分类文章列表
export async function fetchArticlesBySectionCode(sectionCode: string): Promise<Article[]> {
  const { payload } = await http.get<{ articles: any[] }>(`/articles/${sectionCode}`)
  return payload.articles.map(articleData => new Article(articleData))
}

// fetchArticle 获取文章详情
export async function fetchArticle(id: number): Promise<Article | undefined> {
  const { payload } = await http.get<{ article: any }>(`/articles/${id}`)
  return payload.article ? new Article(payload.article) : undefined
}
