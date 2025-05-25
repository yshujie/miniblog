import type { Section } from './section'

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
}