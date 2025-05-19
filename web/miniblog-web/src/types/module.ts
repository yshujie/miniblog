import type { Section } from './section'

// Module 模块
export interface Module {
  id: number
  title: string
  code: string
  sections: Section[]
}