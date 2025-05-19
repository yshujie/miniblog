import type { Article } from './article'

// Section 章节
export interface Section {
  moduleCode: string
  id: number
  title: string
  code: string
  articles: Article[]
}