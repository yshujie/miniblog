import type { Section } from '../types/section'
import http from '../util/http'

// fetchSectionsByModuleCode 获取模块下的所有章节
export async function fetchSectionsByModuleCode(moduleCode: string): Promise<Section[]> {
  const { data } = await http.get<Section[]>(`/sections?module_code=${moduleCode}`)
  return data
}

// fetchSectionByCode 获取章节详情
export async function fetchSectionByCode(code: string): Promise<Section | undefined> {
  const { data } = await http.get<Section>(`/sections/${code}`)
  return data
}