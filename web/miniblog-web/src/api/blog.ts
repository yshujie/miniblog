import http from '@/util/http'
import { Module } from '../types/module'

// fetchModuleDetail 获取模块详情
export async function fetchModuleDetail(moduleCode: string): Promise<Module> {
  console.log('fetchModuleDetail', moduleCode)
  const { payload } = await http.get<{ moduleDetail: any }>(`/blog/moduleDetail?module_code=${moduleCode}`)
  console.log('payload', payload)
  return new Module(payload.moduleDetail)
}