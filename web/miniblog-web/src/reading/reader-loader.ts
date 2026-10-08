import { reactive } from 'vue'
import type { LocationQuery, RouteLocationRaw } from 'vue-router'
import type { Article } from '@/types/article'
import type { Module } from '@/types/module'
import { decimalID } from '@/api/reading-contract'
import { ApiError } from '@/util/http'
import { containsPlacement, firstArticle } from '@/util/reading'

export type ReaderStatus = 'loading' | 'success' | 'empty' | 'not_found' | 'error'
export interface ReadingLocation {
  moduleCode: string
  articleId?: string
  query?: LocationQuery
  hash?: string
}
interface Dependencies {
  loadModule: (code: string, force?: boolean) => Promise<Module>
  loadArticle: (id: string, signal: AbortSignal) => Promise<Article>
  replace: (target: RouteLocationRaw) => Promise<unknown>
}

function waitForShared<T>(request: Promise<T>, signal: AbortSignal): Promise<T> {
  return new Promise((resolve, reject) => {
    const cancel = () => reject(new DOMException('Navigation cancelled', 'AbortError'))
    if (signal.aborted) { cancel(); return }
    signal.addEventListener('abort', cancel, { once: true })
    request.then(value => {
      signal.removeEventListener('abort', cancel)
      resolve(value)
    }, error => {
      signal.removeEventListener('abort', cancel)
      reject(error)
    })
  })
}

export function createReaderLoader(deps: Dependencies) {
  const state = reactive({
    status: 'loading' as ReaderStatus,
    busy: false,
    module: null as Module | null,
    article: null as Article | null,
    message: '',
    refreshError: '',
    resource: 'module' as 'module' | 'article',
  })
  let version = 0
  let controller: AbortController | undefined
  // Carry only the freshly resolved article across its canonical replace.
  let canonicalArticle: { code: string; id: string; article: Article } | undefined

  async function load(location: ReadingLocation, force = false, background = false): Promise<void> {
    if (background && state.busy) return
    const retain = background && state.status === 'success' && state.article?.id === location.articleId && state.module?.code === location.moduleCode
    const current = ++version
    controller?.abort()
    controller = new AbortController()
    const signal = controller.signal
    const latest = () => current === version && !signal.aborted
    const carried = canonicalArticle
    canonicalArticle = undefined
    if (!retain) {
      state.status = 'loading'
      state.article = null
      state.module = null
    }
    state.busy = true
    state.refreshError = ''
    state.message = ''
    state.resource = location.articleId ? 'article' : 'module'
    try {
      if (location.articleId) {
        let id: string
        try { id = decimalID(location.articleId) } catch {
          if (latest()) { state.status = 'not_found'; state.message = '文章链接中的 ID 无效' }
          return
        }
        const article = carried?.id === id && carried.code === location.moduleCode
          ? carried.article : await deps.loadArticle(id, signal)
        if (!latest()) return
        // module_code is additive; fall back to the historical module on an older backend.
        const code = article.moduleCode || location.moduleCode
        if (code !== location.moduleCode || article.id !== location.articleId) {
          canonicalArticle = { code, id: article.id, article }
          await deps.replace({
            name: 'BlogArticle', params: { module: code, article: article.id },
            query: location.query, hash: location.hash,
          })
          return
        }
        state.resource = 'module'
        let module = await waitForShared(deps.loadModule(code, force), signal)
        if (!latest()) return
        if (!containsPlacement(module, article) && !force) {
          module = await waitForShared(deps.loadModule(code, true), signal)
          if (!latest()) return
        }
        state.module = module
        state.article = article
        state.status = 'success'
      } else {
        const module = await waitForShared(deps.loadModule(location.moduleCode, true), signal)
        if (!latest()) return
        state.module = module
        const first = firstArticle(module)
        if (!first) {
          state.status = 'empty'
          return
        }
        await deps.replace({
          name: 'BlogArticle', params: { module: module.code, article: first.id },
          query: location.query, hash: location.hash,
        })
      }
    } catch (error) {
      if (!latest()) return
      const missing = error instanceof ApiError && error.status === 404
      if (retain && !missing) {
        state.refreshError = '更新失败，正在显示上次成功读取的内容。请检查网络后重试'
        return
      }
      state.article = null
      state.module = null
      state.status = missing ? 'not_found' : 'error'
      state.message = state.status === 'not_found'
        ? (state.resource === 'article' ? '文章不存在或已下架' : '模块不存在或不可用')
        : '加载失败，请检查网络后重试'
    } finally {
      if (latest()) state.busy = false
    }
  }

  function dispose() {
    version++
    controller?.abort()
    canonicalArticle = undefined
  }

  return { state, load, dispose }
}
