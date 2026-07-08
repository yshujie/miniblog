import type { Article } from './article'

export class Subsection {
  id: string
  sectionCode: string
  title: string
  code: string
  articles: Article[]

  constructor(data: {
    id: string
    sectionCode: string
    title: string
    code: string
    articles?: Article[]
  }) {
    this.id = data.id
    this.sectionCode = data.sectionCode
    this.title = data.title
    this.code = data.code
    this.articles = data.articles || []
  }
}
