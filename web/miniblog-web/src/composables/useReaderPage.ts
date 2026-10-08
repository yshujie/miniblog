import { onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useModuleStore } from '@/stores/module'
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
    query: route.query,
    hash: route.hash,
  })
  watch(() => [route.params.module, route.params.article], () => {
    void reader.load(location())
  }, { immediate: true })
  onUnmounted(reader.dispose)
  return { state: reader.state, retry: () => reader.load(location(), true) }
}
