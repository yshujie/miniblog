import { onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useModuleStore, READING_TTL_MS } from '@/stores/module'
import { fetchArticleDetail } from '@/api/blog'
import { createReaderLoader } from '@/reading/reader-loader'

export function useReaderPage() {
  const route = useRoute()
  const router = useRouter()
  const store = useModuleStore()
  const reader = createReaderLoader({
    loadModule: (code, force) => store.loadModuleDetail(code, force),
    loadArticle: fetchArticleDetail,
    replace: target => router.replace(target),
  })
  const location = () => ({
    moduleCode: String(route.params.module || ''),
    articleId: typeof route.params.article === 'string' ? route.params.article : undefined,
    query: route.query, hash: route.hash,
  })
  let disposed = false
  let timer: ReturnType<typeof setInterval> | undefined
  let resume: ReturnType<typeof setTimeout> | undefined
  const refresh = () => {
    if (disposed || document.visibilityState === 'hidden' || reader.state.busy) return
    void reader.load(location(), true, true)
    void store.loadModules(true).catch(() => {})
  }
  function updateVisibility() {
    clearInterval(timer)
    timer = undefined
    if (document.visibilityState === 'hidden') { clearTimeout(resume); resume = undefined; return }
    timer = setInterval(refresh, READING_TTL_MS)
    // Focus and visibility events usually arrive together; issue one refresh.
    clearTimeout(resume)
    resume = setTimeout(refresh, 50)
  }
  const onFocus = () => { if (document.visibilityState !== 'hidden') { clearTimeout(resume); resume = setTimeout(refresh, 50) } }
  watch(() => [route.params.module, route.params.article], () => {
    clearTimeout(resume)
    void reader.load(location())
  }, { immediate: true })
  if (document.visibilityState !== 'hidden') timer = setInterval(refresh, READING_TTL_MS)
  document.addEventListener('visibilitychange', updateVisibility)
  window.addEventListener('focus', onFocus)
  onUnmounted(() => {
    disposed = true
    clearInterval(timer); clearTimeout(resume)
    document.removeEventListener('visibilitychange', updateVisibility)
    window.removeEventListener('focus', onFocus)
    reader.dispose()
  })
  return { state: reader.state, retry: () => reader.load(location(), true, Boolean(reader.state.article)) }
}
