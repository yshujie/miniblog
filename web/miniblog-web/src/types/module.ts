import type { Section } from './section'

// Module 模块
export class Module {
  id: string
  title: string
  code: string
  sections: Section[]

  constructor(data: {
    id: string
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