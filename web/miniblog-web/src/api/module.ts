import type { Module } from '../types/module'
import { fetchSectionsByModuleCode } from './section'

// mock 数据
const mockModules: Module[] = [
  {
    id: 1,
    title: 'AI',
    code: 'ai',
    sections: [],
  },
  {
    id: 2,
    title: 'Golang',
    code: 'golang',
    sections: []
  },
  {
    id: 3,
    title: '架构设计',
    code: 'architecture',
    sections: []
  },
  {
    id: 4,
    title: '编程',
    code: 'programming',
    sections: []
  },
]

// fetchModules 获取所有模块
export async function fetchModules(): Promise<Module[]> {
  return new Promise(resolve => setTimeout(() => resolve(mockModules), 300))
}

// fetchModuleByCode 获取模块详情
export async function fetchModuleByCode(code: string): Promise<Module | undefined> {
  return new Promise(resolve => setTimeout(async () => {
    console.log(`in fetchModuleByCode, code: ${code}`)

    // 查找 module 数据
    var fetchedModule = mockModules.find(m => m.code === code)

    // 查找 sections 数据
    if (fetchedModule && fetchedModule.sections.length === 0) {
      fetchedModule.sections = await fetchSectionsByModuleCode(code)
    }

    resolve(fetchedModule)
  }, 300))
}