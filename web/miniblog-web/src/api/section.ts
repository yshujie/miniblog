import { Section } from '@/types/section'
import http from '@/util/http'

// fetchSectionsByModuleCode 获取模块下的所有章节
export async function fetchSectionsByModuleCode(moduleCode: string): Promise<Section[]> {
  const { payload } = await http.get<{ sections: any[] }>(`/sections/${moduleCode}`)
  return payload.sections.map(sectionData => new Section(sectionData))
}

// fetchSectionByCode 获取章节详情
export async function fetchSectionByCode(code: string): Promise<Section | undefined> {
  const { payload } = await http.get<{ section: any }>(`/sections/${code}`)
  return payload.section ? new Section(payload.section) : undefined
}