import type { Article } from './article'
import { fetchArticlesBySectionCode } from '@/api/article'

// Section 章节
export class Section {
  moduleCode: string
  id: number
  title: string
  code: string
  articles: Article[]

  constructor(data: {
    moduleCode: string
    id: number
    title: string
    code: string
    articles?: Article[]
  }) {
    this.moduleCode = data.moduleCode
    this.id = data.id
    this.title = data.title
    this.code = data.code
    this.articles = data.articles || []
  }

  // 加载章节下的所有文章
  async loadArticles(): Promise<Article[]> {
    const articles = await fetchArticlesBySectionCode(this.code)
    this.articles = articles
    return articles
  }
}