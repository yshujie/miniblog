import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { defineComponent, h } from 'vue'
import { useReaderPage } from '../useReaderPage'
import { makeArticle, makeModule } from '@/__tests__/fixtures'
const { article, module, summaries } = vi.hoisted(() => ({ article: vi.fn(), module: vi.fn(), summaries: vi.fn() }))
vi.mock('@/api/blog', () => ({ fetchArticleDetail: article, fetchModuleDetail: module }))
vi.mock('@/api/module', () => ({ fetchModules: summaries }))
let wrapper: VueWrapper | undefined
beforeEach(() => {
  vi.useFakeTimers(); vi.resetAllMocks()
  Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'visible' })
  article.mockResolvedValue(makeArticle()); module.mockResolvedValue(makeModule()); summaries.mockResolvedValue([makeModule()])
})
afterEach(() => { wrapper?.unmount(); wrapper = undefined; vi.useRealTimers(); vi.restoreAllMocks() })
describe('visible reader refresh lifecycle', () => {
  it('refreshes at sixty seconds, coalesces resume events, and stops after unmount', async () => {
    const page = defineComponent({ setup() { useReaderPage(); return () => h('div') } })
    const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/blog/:module/article/:article', component: page }] })
    await router.push('/blog/go/article/9007199254740993')
    wrapper = mount({ template: '<router-view />' }, { global: { plugins: [createPinia(), router] } })
    await flushPromises(); expect(article).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(59999); expect(article).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(1); await flushPromises(); expect(article).toHaveBeenCalledTimes(2)
    Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'hidden' })
    document.dispatchEvent(new Event('visibilitychange'))
    await vi.advanceTimersByTimeAsync(60000); expect(article).toHaveBeenCalledTimes(2)
    Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'visible' })
    document.dispatchEvent(new Event('visibilitychange')); window.dispatchEvent(new Event('focus'))
    await vi.advanceTimersByTimeAsync(50); await flushPromises(); expect(article).toHaveBeenCalledTimes(3)
    wrapper.unmount(); wrapper = undefined
    window.dispatchEvent(new Event('focus')); await vi.advanceTimersByTimeAsync(60000)
    expect(article).toHaveBeenCalledTimes(3)
  })
})
