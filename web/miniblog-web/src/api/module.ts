import http from '@/util/http'
import { Module } from '../types/module'

// fetchModules 获取所有模块
export async function fetchModules(): Promise<Module[]> {
  const { payload } = await http.get<{ modules: any[] }>('/blog/modules')
  return payload.modules.map(moduleData => new Module(moduleData))
}
