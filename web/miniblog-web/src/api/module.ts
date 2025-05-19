import http from '@/util/http'
import type { Module } from '../types/module'

// fetchModules 获取所有模块
export async function fetchModules(): Promise<Module[]> {
  const { data } = await http.get<Module[]>('/modules')
  return data
}

// fetchModuleByCode 获取模块详情
export async function fetchModuleByCode(code: string): Promise<Module | undefined> {
  const { data } = await http.get<Module>(`/modules/${code}`)
  return data
}