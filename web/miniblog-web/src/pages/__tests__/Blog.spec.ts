import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import Blog from '../Blog.vue'
import { ApiError } from '@/util/http'
import { makeArticle, makeModule } from '@/__tests__/fixtures'
import { useUiStore } from '@/stores/ui'
const { detail, article } = vi.hoisted(() => ({ detail: vi.fn(), article: vi.fn() }))
vi.mock('@/api/blog', () => ({ fetchModuleDetail: detail, fetchArticleDetail: article }))
vi.mock('@/api/module', () => ({ fetchModules: vi.fn().mockResolvedValue([]) }))
beforeEach(() => { detail.mockReset(); article.mockReset() })
afterEach(() => { vi.useRealTimers(); vi.restoreAllMocks(); vi.unstubAllGlobals() })
async function mountPage(path: string) {
  const router = createRouter({ history: createMemoryHistory(), routes: [
    { path: '/blog/:module', name: 'BlogModule', component: Blog },
    { path: '/blog/:module/article/:article', name: 'BlogArticle', component: Blog },
    { path: '/', component: { template: '<div>首页</div>' } },
  ] })
  await router.push(path)
  const pinia = createPinia()
  const wrapper = mount({ template: '<router-view />' }, { global: { plugins: [pinia, router], stubs: {
    ElDrawer: { props: ['modelValue'], template: '<div v-if="modelValue" role="dialog"><slot /></div>' },
    ElButton: { template: '<button><slot /></button>' },
  } } })
  await flushPromises()
  return { wrapper, router, pinia }
}
describe('reading page routes', () => {
  it('keeps one iframe while directory panels, viewport and refreshed metadata change', async () => {
    vi.useFakeTimers()
    vi.stubGlobal('innerWidth', 1440)
    const initial = makeArticle()
    article.mockResolvedValue(initial)
    detail.mockResolvedValue(makeModule())
    const { wrapper, pinia } = await mountPage('/blog/go/article/9007199254740993')
    const frame = wrapper.get('iframe').element
    const ui = useUiStore(pinia)
    ui.toggleSidebar()
    await flushPromises()
    expect(wrapper.get('iframe').element).toBe(frame)
    vi.stubGlobal('innerWidth', 390)
    window.dispatchEvent(new Event('resize'))
    ui.openDrawer()
    await flushPromises()
    expect(wrapper.get('iframe').element).toBe(frame)
    ui.openTopics()
    await flushPromises()
    expect(ui.mobileDrawerOpen).toBe(false)
    expect(wrapper.get('iframe').element).toBe(frame)
    ui.closePanels()
    const refreshed = { ...initial, title: '刷新后的长标题', author: 'Shujie', tags: ['Go'] }
    article.mockResolvedValue(refreshed)
    await vi.advanceTimersByTimeAsync(60000)
    await flushPromises()
    expect(wrapper.get('iframe').element).toBe(frame)
    expect(wrapper.findAll('iframe')).toHaveLength(1)
    expect(wrapper.get('h1').text()).toBe(refreshed.title)
    vi.stubGlobal('innerWidth', 1440)
    window.dispatchEvent(new Event('resize'))
    await flushPromises()
    expect(wrapper.get('iframe').element).toBe(frame)
    wrapper.unmount()
    vi.unstubAllGlobals()
  })
  it('opens a moved article when the bookmarked module no longer exists', async () => {
    article.mockResolvedValue(makeArticle('9007199254740993', 'new'))
    detail.mockResolvedValue(makeModule('new'))
    const { wrapper, router } = await mountPage('/blog/deleted/article/9007199254740993?from=old#part')
    expect(router.currentRoute.value.fullPath).toBe('/blog/new/article/9007199254740993?from=old#part')
    expect(article).toHaveBeenCalledTimes(1)
    expect(detail).toHaveBeenCalledWith('new')
    expect(wrapper.find('iframe').exists()).toBe(true)
    wrapper.unmount()
  })
  it('shows specific article absence and retries without selecting a different article', async () => {
    article.mockRejectedValueOnce(new ApiError('missing', 404)).mockResolvedValueOnce(makeArticle())
    detail.mockResolvedValue(makeModule())
    const { wrapper, router } = await mountPage('/blog/go/article/9007199254740993')
    expect(wrapper.get('[role="alert"]').text()).toContain('文章不可用')
    expect(wrapper.find('iframe').exists()).toBe(false)
    await wrapper.get('[role="alert"] button').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.params.article).toBe('9007199254740993')
    expect(wrapper.find('iframe').exists()).toBe(true)
    wrapper.unmount()
  })
  it('leaves an empty module on its URL with an explicit empty state', async () => {
    detail.mockResolvedValue(makeModule('empty', []))
    const { wrapper, router } = await mountPage('/blog/empty')
    expect(wrapper.text()).toContain('暂无已发布文章')
    expect(router.currentRoute.value.fullPath).toBe('/blog/empty')
    expect(article).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
