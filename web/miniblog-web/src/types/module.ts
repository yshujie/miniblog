import type { Section } from './section'
import { fetchSectionsByModuleCode } from '@/api/section'

// Module 模块
export class Module {
  id: number
  title: string
  code: string
  sections: Section[]

  constructor(data: {
    id: number
    title: string
    code: string
    sections?: Section[]
  }) {
    this.id = data.id
    this.title = data.title
    this.code = data.code
    this.sections = data.sections || []
  }

  // 加载模块下的所有章节
  async loadSections(): Promise<Section[]> {
    const sections = await fetchSectionsByModuleCode(this.code)
    this.sections = sections
    return sections
  }
}