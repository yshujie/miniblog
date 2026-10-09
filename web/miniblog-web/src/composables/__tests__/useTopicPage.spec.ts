import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { defineComponent, h } from 'vue'
import { createRouter, createMemoryHistory } from 'vue-router'
import { useTopicPage } from '../useTopicPage'
import { deferred, makeModule } from '@/__tests__/fixtures'
import { ApiError } from '@/util/http'
import type { Module } from '@/types/module'

const { detail, article, summaries } = vi.hoisted(() => ({ detail: vi.fn(), article: vi.fn(), summaries: vi.fn() }))
vi.mock('@/api/blog', () => ({ fetchModuleDetail: detail, fetchArticleDetail: article }))
vi.mock('@/api/module', () => ({ fetchModules: summaries }))
let wrapper: VueWrapper | undefined
let topic: ReturnType<typeof useTopicPage>

beforeEach(() => {
  vi.useFakeTimers()
  vi.resetAllMocks()
  Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'visible' })
  detail.mockImplementation((code: string) => Promise.resolve(makeModule(code)))
  summaries.mockResolvedValue([makeModule()])
})
afterEach(() => { wrapper?.unmount(); wrapper = undefined; vi.useRealTimers(); vi.restoreAllMocks() })

async function openTopic(code = 'go') {
  const page = defineComponent({ setup() { topic = useTopicPage(); return () => h('div') } })
  const router = createRouter({ history: createMemoryHistory(), routes: [
    { path: '/topics/:module', name: 'TopicOverview', component: page },
    { path: '/blog/:module', name: 'BlogModule', component: { template: '<div />' } },
  ] })
  await router.push(`/topics/${code}`)
  wrapper = mount({ template: '<router-view />' }, { global: { plugins: [createPinia(), router] } })
  await flushPromises()
  return router
}

describe('independent public topic loading', () => {
  it('loads only the public directory without requesting an article or redirecting to the first article', async () => {
    const router = await openTopic()
    expect(topic.state.status).toBe('success')
    expect(topic.state.module?.code).toBe('go')
    expect(detail).toHaveBeenCalledTimes(1)
    expect(article).not.toHaveBeenCalled()
    expect(router.currentRoute.value.fullPath).toBe('/topics/go')
  })

  it('ignores late success after switching themes while permitting the shared cache request to finish', async () => {
    const old = deferred<Module>()
    detail.mockImplementation((code: string) => code === 'go' ? old.promise : Promise.resolve(makeModule(code)))
    const router = await openTopic()
    expect(topic.state.status).toBe('loading')
    await router.push('/topics/ddd')
    await flushPromises()
    expect(topic.state.module?.code).toBe('ddd')
    old.resolve(makeModule('go'))
    await flushPromises()
    expect(topic.state.status).toBe('success')
    expect(topic.state.module?.code).toBe('ddd')
    expect(topic.state.busy).toBe(false)
  })

  it('ignores an old request failure after a newer topic has loaded', async () => {
    const old = deferred<Module>()
    detail.mockImplementation((code: string) => code === 'go' ? old.promise : Promise.resolve(makeModule(code)))
    const router = await openTopic()
    await router.push('/topics/ddd')
    await flushPromises()
    old.reject(new ApiError('gone', 404))
    await flushPromises()
    expect(topic.state.status).toBe('success')
    expect(topic.state.module?.code).toBe('ddd')
    expect(topic.state.message).toBe('')
  })

  it('retains a successfully shown directory on temporary refresh failure, then clears it on authoritative 404', async () => {
    await openTopic()
    const shown = topic.state.module
    detail.mockRejectedValueOnce(new Error('offline'))
    await topic.retry()
    expect(topic.state.status).toBe('success')
    expect(topic.state.module).toBe(shown)
    expect(topic.state.refreshError).toContain('上次成功读取')
    detail.mockRejectedValueOnce(new ApiError('gone', 404))
    await topic.retry()
    expect(topic.state.status).toBe('not_found')
    expect(topic.state.module).toBeNull()
    expect(topic.state.refreshError).toBe('')
    expect(topic.state.message).toContain('主题不存在')
  })

  it('keeps an empty theme at its address and retains its verified empty state on temporary refresh failure', async () => {
    detail.mockResolvedValue(makeModule('go', []))
    const router = await openTopic()
    expect(topic.state.status).toBe('empty')
    expect(router.currentRoute.value.fullPath).toBe('/topics/go')
    detail.mockRejectedValueOnce(new Error('offline'))
    await topic.retry()
    expect(topic.state.status).toBe('empty')
    expect(topic.state.module?.title).toBe('go')
    expect(topic.state.refreshError).toContain('更新失败')
  })

  it('distinguishes initial service errors from missing themes and supports retrying the same target', async () => {
    detail.mockRejectedValueOnce(new Error('offline'))
    const router = await openTopic()
    expect(topic.state.status).toBe('error')
    expect(topic.state.module).toBeNull()
    await topic.retry()
    expect(topic.state.status).toBe('success')
    expect(router.currentRoute.value.fullPath).toBe('/topics/go')
    expect(article).not.toHaveBeenCalled()
  })

  it('refreshes after sixty visible seconds, coalesces visibility/focus recovery, and stops after unmount', async () => {
    await openTopic()
    await vi.advanceTimersByTimeAsync(59999)
    expect(detail).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(1)
    await flushPromises()
    expect(detail).toHaveBeenCalledTimes(2)
    Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'hidden' })
    document.dispatchEvent(new Event('visibilitychange'))
    await vi.advanceTimersByTimeAsync(60000)
    expect(detail).toHaveBeenCalledTimes(2)
    Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'visible' })
    document.dispatchEvent(new Event('visibilitychange'))
    window.dispatchEvent(new Event('focus'))
    await vi.advanceTimersByTimeAsync(50)
    await flushPromises()
    expect(detail).toHaveBeenCalledTimes(3)
    wrapper?.unmount()
    wrapper = undefined
    window.dispatchEvent(new Event('focus'))
    document.dispatchEvent(new Event('visibilitychange'))
    await vi.advanceTimersByTimeAsync(60000)
    expect(detail).toHaveBeenCalledTimes(3)
  })

  it('does not reload for chapter/query changes and ignores a request completing after unmount', async () => {
    const router = await openTopic()
    await router.push('/topics/go?chapter=s&view=all')
    await flushPromises()
    expect(detail).toHaveBeenCalledTimes(1)
    const pending = deferred<Module>()
    detail.mockReturnValueOnce(pending.promise)
    const refresh = topic.retry()
    wrapper?.unmount()
    wrapper = undefined
    pending.resolve(makeModule('changed'))
    await refresh
    expect(topic.state.module?.code).toBe('go')
  })
})
