// Article 文章
export class Article {
  id: number
  sectionCode: string
  title: string
  content: string
  externalUrl: string
  author: string
  tags: string[]
  createdAt: string
  updatedAt: string

  constructor(data: {
    id: number
    sectionCode: string
    title: string
    externalUrl: string
    author: string
    content: string | undefined
    tags: string[] | undefined
    createdAt: string | undefined
    updatedAt: string | undefined
  }) {
    this.id = data.id
    this.sectionCode = data.sectionCode
    this.title = data.title
    this.externalUrl = data.externalUrl || ''
    this.author = data.author
    this.content = data.content || ''
    this.tags = data.tags || []
    this.createdAt = data.createdAt || ''
    this.updatedAt = data.updatedAt || ''
  }

  // 获取文章的 markdown 内容
  getMarkdownContent(): string {
    return this.content
  }
}
