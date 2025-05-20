import http from '@/util/http'
import { Module } from '../types/module'

// fetchModules 获取所有模块
export async function fetchModules(): Promise<Module[]> {
  const { payload } = await http.get<{ modules: any[] }>('/modules')
  return payload.modules.map(moduleData => new Module(moduleData))
}

// fetchModuleByCode 获取模块详情
export async function fetchModuleByCode(code: string): Promise<Module | undefined> {
  const { payload } = await http.get<{ module: any }>(`/modules/${code}`)
  return payload.module ? new Module(payload.module) : undefined
}