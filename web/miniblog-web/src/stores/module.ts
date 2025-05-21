import { defineStore } from 'pinia'
import type { Module } from '@/types/module'
import { fetchModules } from '@/api/module'
import { Section } from '@/types/section'
import { fetchModuleDetail } from '@/api/blog'
import { Article } from '@/types/article'

// module store
export const useModuleStore = defineStore('module', {
  state: () => ({
    currentModule: null as Module | null,
    modules: [] as Module[],
  }),

  getters: {
    // 获取所有模块
    getAllModules: (state): Module[] => state.modules,

    // 根据 code 获取模块
    getModuleByCode: (state) => (code: string): Module | undefined => {
      return state.modules.find(module => module.code === code)
    },

    // 根据 code 获取模块下的所有章节
    getSectionsByCode: (state) => (code: string): Section[] => {
      const module = state.modules.find(module => module.code === code)
      return module?.sections || []
    },
  },

  actions: {
    // 加载所有模块
    async loadModules() {
      if (this.modules.length > 0) {
        return // 如果已经加载过，直接返回
      }

      this.modules = await fetchModules()
    },

    // 加载模块详情
    async loadAllModuleDetail() {
      for (const module of this.modules) {
        const moduleDetail = await fetchModuleDetail(module.code)
        module.title = moduleDetail.title
        module.sections = moduleDetail.sections.map(section => new Section(section))
        module.sections.forEach(section => {
          section.articles = section.articles.map(article => new Article(article))
        })
      }
    },

    // 设置当前模块
    setCurrentModule(module: Module) {
      this.currentModule = module
    },

    // 清除当前模块
    clearCurrentModule() {
      this.currentModule = null
    },

    // 清除模块数据
    clearModules() {
      this.modules = []
    }
  }
}) 