import { onMounted, onUnmounted, reactive, watch } from 'vue'
import { useModuleStore, READING_TTL_MS } from '@/stores/module'
import type { Module } from '@/types/module'

interface TopicPreview { status: 'idle' | 'loading' | 'success' | 'error'; module: Module | null; error: string }

export function useHomeTopics() {
  const store = useModuleStore()
  const previews = reactive<Record<string, TopicPreview>>(Object.create(null))
  const elements = new Map<string, Element>()
  const visible = new Set<string>()
  const queue: { code: string; force: boolean }[] = []
  const pending = new Set<string>()
  let observer: IntersectionObserver | undefined
  let active = 0, mounted = false, disposed = false
  const exists = (code: string) => store.modules.some(topic => topic.code === code)

  function drain() {
    while (!disposed && active < 3 && queue.length) {
      const item = queue.shift()!
      if (!exists(item.code)) { pending.delete(item.code); continue }
      active++
      void store.loadModuleDetail(item.code, item.force).then(module => {
        if (!disposed && exists(item.code) && store.directoryCache[item.code]?.completeness === 'full') {
          previews[item.code] = { status: 'success', module, error: '' }
        }
      }, () => {
        if (!disposed && exists(item.code)) {
          previews[item.code] = { status: 'error', module: null, error: '章节预览暂时无法加载。' }
        }
      }).finally(() => { active--; pending.delete(item.code); drain() })
    }
  }
  function requestTopic(code: string, force = false) {
    if (disposed || !exists(code) || pending.has(code)) return
    const cached = store.directoryCache[code]
    if (!force && cached?.completeness === 'full' && Date.now() - cached.fetchedAt < READING_TTL_MS) {
      previews[code] = { status: 'success', module: cached.module, error: '' }
      return
    }
    previews[code] = { status: 'loading', module: null, error: '' }
    pending.add(code); queue.push({ code, force }); drain()
  }
  function registerTopic(element: unknown, code: string) {
    const previous = elements.get(code)
    if (!(element instanceof Element)) {
      if (previous) observer?.unobserve(previous)
      elements.delete(code)
      return
    }
    if (previous === element) return
    if (previous) observer?.unobserve(previous)
    elements.set(code, element)
    observer?.observe(element)
    if (mounted && !observer && store.modules.slice(0, 3).some(topic => topic.code === code)) requestTopic(code)
  }
  watch(() => store.modules.map(topic => topic.code), codes => {
    const current = new Set(codes)
    for (const code of Object.keys(previews)) {
      if (current.has(code)) continue
      delete previews[code]; visible.delete(code)
      const element = elements.get(code)
      if (element) observer?.unobserve(element)
      elements.delete(code)
    }
    // Remove only queued work; already shared in-flight requests keep their owner.
    for (let index = queue.length - 1; index >= 0; index--) {
      if (!current.has(queue[index].code)) {
        pending.delete(queue[index].code)
        queue.splice(index, 1)
      }
    }
    for (const code of codes) {
      const cached = store.directoryCache[code]
      if (cached?.completeness === 'full') previews[code] = { status: 'success', module: cached.module, error: '' }
      else if (!previews[code]) previews[code] = { status: 'idle', module: null, error: '' }
    }
    if (mounted && !observer) codes.slice(0, 3).forEach(code => requestTopic(code))
  }, { immediate: true })
  onMounted(() => {
    mounted = true
    if (typeof IntersectionObserver === 'function') {
      observer = new IntersectionObserver(entries => {
        if (disposed) return
        for (const entry of entries) {
          if (!entry.isIntersecting) continue
          const code = [...elements].find(([, element]) => element === entry.target)?.[0]
          if (!code || visible.has(code)) continue
          visible.add(code); observer?.unobserve(entry.target); requestTopic(code)
        }
      }, { rootMargin: '200px 0px' })
      for (const element of elements.values()) observer.observe(element)
    } else store.modules.slice(0, 3).forEach(topic => requestTopic(topic.code))
    void store.loadModules().catch(() => {})
  })
  onUnmounted(() => { disposed = true; observer?.disconnect(); elements.clear(); visible.clear(); queue.length = 0 })
  const retryTopic = (code: string) => requestTopic(code, true)
  const retryList = () => { void store.loadModules(true).catch(() => {}) }
  return { store, previews, registerTopic, retryTopic, retryList }
}
