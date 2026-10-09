import { onUnmounted, reactive, watch } from 'vue'
import { useRoute } from 'vue-router'
import type { Module } from '@/types/module'
import { useModuleStore, READING_TTL_MS } from '@/stores/module'
import { ApiError } from '@/util/http'
import { flattenCatalog } from '@/util/catalog'

export type TopicStatus = 'loading' | 'success' | 'empty' | 'not_found' | 'error'

export function useTopicPage() {
  const route = useRoute()
  const store = useModuleStore()
  const state = reactive({
    status: 'loading' as TopicStatus,
    busy: false,
    module: null as Module | null,
    message: '',
    refreshError: '',
  })
  let version = 0
  let disposed = false
  let timer: ReturnType<typeof setInterval> | undefined
  let resume: ReturnType<typeof setTimeout> | undefined
  const moduleCode = () => String(route.params.module || '')

  async function load(force = false, background = false) {
    const code = moduleCode()
    const current = ++version
    const latest = () => !disposed && current === version
    const retain = background && state.module?.code === code && ['success', 'empty'].includes(state.status)
    if (!retain) {
      state.status = 'loading'
      state.module = null
    }
    state.busy = true
    state.message = ''
    state.refreshError = ''
    try {
      // This request is shared by the store: leaving the page must not cancel it for other consumers.
      const module = await store.loadModuleDetail(code, force)
      if (!latest()) return
      state.module = module
      state.status = flattenCatalog(module).length ? 'success' : 'empty'
    } catch (error) {
      if (!latest()) return
      const missing = error instanceof ApiError && error.status === 404
      if (retain && !missing) {
        state.refreshError = '更新失败，正在显示上次成功读取的目录。请检查网络后重试'
        return
      }
      state.module = null
      state.status = missing ? 'not_found' : 'error'
      state.message = missing ? '主题不存在或不可用' : '加载失败，请检查网络后重试'
    } finally {
      if (latest()) state.busy = false
    }
  }

  const refresh = () => {
    if (disposed || document.visibilityState === 'hidden' || state.busy) return
    void load(true, true)
    void store.loadModules(true).catch(() => {})
  }
  function updateVisibility() {
    clearInterval(timer)
    timer = undefined
    clearTimeout(resume)
    resume = undefined
    if (document.visibilityState === 'hidden') return
    timer = setInterval(refresh, READING_TTL_MS)
    resume = setTimeout(refresh, 50)
  }
  const onFocus = () => {
    if (document.visibilityState === 'hidden') return
    clearTimeout(resume)
    resume = setTimeout(refresh, 50)
  }
  watch(() => route.params.module, () => {
    clearTimeout(resume)
    void load()
  }, { immediate: true })
  if (document.visibilityState !== 'hidden') timer = setInterval(refresh, READING_TTL_MS)
  document.addEventListener('visibilitychange', updateVisibility)
  window.addEventListener('focus', onFocus)
  onUnmounted(() => {
    disposed = true
    version++
    clearInterval(timer)
    clearTimeout(resume)
    document.removeEventListener('visibilitychange', updateVisibility)
    window.removeEventListener('focus', onFocus)
  })
  return { state, retry: () => load(true, Boolean(state.module)) }
}
