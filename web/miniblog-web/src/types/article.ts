// Article 文章
export class Article {
  id: string
  moduleCode: string
  sectionCode: string
  subsectionCode: string
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
    moduleCode?: string
    sectionCode?: string
    subsectionCode?: string
    title: string
    externalLink?: string
    author?: string
    content?: string
    tags?: string[]
    pos?: number
    createdAt?: string
    updatedAt?: string
  }) {
    this.id = data.id
    this.moduleCode = data.moduleCode || ''
    this.sectionCode = data.sectionCode || ''
    this.subsectionCode = data.subsectionCode || ''
    this.title = data.title
    this.author = data.author || ''
    this.externalLink = data.externalLink || ''
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
