// Article 文章
export class Article {
  id: string
  sectionCode: string
  title: string
  content: string
  externalLink: string
  author: string
  tags: string[]
  pos: number
  createdAt: string
  updatedAt: string

  constructor(data: {
    id: string
    sectionCode: string
    title: string
    externalLink: string
    author: string
    content: string | undefined
    tags: string[] | undefined
    pos: number | undefined
    createdAt: string | undefined
    updatedAt: string | undefined
  }) {
    this.id = data.id
    this.sectionCode = data.sectionCode
    this.title = data.title
    this.author = data.author
    this.externalLink = data.externalLink
    this.content = data.content || ''
    this.tags = data.tags || []
    this.pos = data.pos || 0
    this.createdAt = data.createdAt || ''
    this.updatedAt = data.updatedAt || ''
  }

  // 获取文章的 markdown 内容
  getMarkdownContent(): string {
    return this.content
  }
}
