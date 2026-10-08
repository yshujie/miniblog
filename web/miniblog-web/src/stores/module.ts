import { defineStore } from 'pinia'
import { computed, reactive, ref } from 'vue'
import type { Module } from '@/types/module'
import { fetchModules } from '@/api/module'
import { fetchModuleDetail } from '@/api/blog'

type ListStatus = 'idle' | 'loading' | 'success' | 'empty' | 'error'
export const READING_TTL_MS = 60000
type CacheEntry = { completeness: 'partial' | 'full'; module: Module; fetchedAt: number }

export const useModuleStore = defineStore('module', () => {
  const modules = ref<Module[]>([])
  const listStatus = ref<ListStatus>('idle')
  const listError = ref('')
  const directoryCache = reactive<Record<string, CacheEntry>>(Object.create(null))
  let listFetchedAt = -Infinity
  let listRequest: Promise<Module[]> | undefined
  const detailRequests = new Map<string, Promise<Module>>()

  function getModuleByCode(code: string) {
    return directoryCache[code]?.module ?? modules.value.find(module => module.code === code)
  }

  async function loadModules(force = false): Promise<Module[]> {
    if (listRequest) return listRequest
    if (!force && Date.now() - listFetchedAt < READING_TTL_MS && ['success', 'empty'].includes(listStatus.value)) return modules.value
    listStatus.value = 'loading'
    listError.value = ''
    listRequest = (async () => {
      try {
        const summaries = await fetchModules()
        modules.value = summaries.map(summary => {
          const cached = directoryCache[summary.code]
          if (cached?.completeness === 'full') {
            cached.module.title = summary.title
            cached.module.id = summary.id || cached.module.id
            return cached.module
          }
          directoryCache[summary.code] = { completeness: 'partial', module: summary, fetchedAt: Date.now() }
          return summary
        })
        const codes = new Set(summaries.map(module => module.code))
        for (const code of Object.keys(directoryCache)) if (!codes.has(code)) delete directoryCache[code]
        listFetchedAt = Date.now()
        listStatus.value = modules.value.length ? 'success' : 'empty'
        return modules.value
      } catch (error) {
        listStatus.value = 'error'
        listError.value = '模块加载失败，请重试'
        throw error
      } finally { listRequest = undefined }
    })()
    return listRequest
  }

  async function loadModuleDetail(code: string, force = false): Promise<Module> {
    const inflight = detailRequests.get(code)
    if (inflight) return inflight
    const cached = directoryCache[code]
    if (!force && cached?.completeness === 'full' && Date.now() - cached.fetchedAt < READING_TTL_MS) return cached.module
    const request = (async () => {
      try {
        const module = await fetchModuleDetail(code)
        directoryCache[code] = { completeness: 'full', module, fetchedAt: Date.now() }
        const index = modules.value.findIndex(item => item.code === code)
        if (index >= 0) modules.value[index] = module
        return module
      } finally { detailRequests.delete(code) }
    })()
    detailRequests.set(code, request)
    return request
  }

  return { modules, listStatus, listError, directoryCache, getModuleByCode, loadModules, loadModuleDetail,
    getAllModules: computed(() => modules.value) }
})
