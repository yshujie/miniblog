import http from '@/util/http'
import type { Module } from '../types/module'

// fetchModules 获取所有模块
export async function fetchModules(): Promise<Module[]> {
  const { payload } = await http.get<{ modules: Module[] }>('/modules')
  return payload.modules
}

// fetchModuleByCode 获取模块详情
export async function fetchModuleByCode(code: string): Promise<Module | undefined> {
  const { payload } = await http.get<{ module: Module }>(`/modules/${code}`)
  return payload.module
}