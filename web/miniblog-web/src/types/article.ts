// Article 文章
export class Article {
  id: number
  sectionCode: string
  title: string
  summary: string
  content: string
  author: string
  tags: string[]
  createdAt: string
  updatedAt: string

  constructor(data: {
    id: number
    sectionCode: string
    title: string
    summary: string
    content: string
    author: string
    tags: string[]
    createdAt: string
    updatedAt: string
  }) {
    this.id = data.id
    this.sectionCode = data.sectionCode
    this.title = data.title
    this.summary = data.summary
    this.content = data.content
    this.author = data.author
    this.tags = data.tags
    this.createdAt = data.createdAt
    this.updatedAt = data.updatedAt
  }

  // 获取文章的 markdown 内容
  getMarkdownContent(): string {
    return this.content
  }
}
