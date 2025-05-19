// Article 文章
export interface Article {
  id: number
  sectionCode: string
  title: string
  summary: string
  content: string
  author: string
  tags: string[]
  createdAt: string
  updatedAt: string
}