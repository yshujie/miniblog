import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import TopicOverview from '../TopicOverview.vue'
import { mapModuleDetail } from '@/api/reading-contract'
import { makeModule } from '@/__tests__/fixtures'
import { ApiError } from '@/util/http'

const { detail, article, summaries } = vi.hoisted(() => ({ detail: vi.fn(), article: vi.fn(), summaries: vi.fn() }))
vi.mock('@/api/blog', () => ({ fetchModuleDetail: detail, fetchArticleDetail: article }))
vi.mock('@/api/module', () => ({ fetchModules: summaries }))
let wrapper: VueWrapper | undefined
const scroll = vi.fn()
const directory = () => mapModuleDetail({
  id: '1', code: 'go', title: 'Golang', sections: [
    { id: '2', code: 'basics', title: 'Go 语言基础', module_code: 'go',
      subsections: [{ id: '3', code: 'types', title: '类型系统', section_code: 'basics',
        articles: [{ id: '9007199254740993', title: 'Go Interface', section_code: 'basics', subsection_code: 'types' }] }],
      articles: [{ id: '9007199254740994', title: '变量与常量', section_code: 'basics', tags: ['Interface'] }] },
    { id: '4', code: 'lifecycle', title: 'Go 生命周期', module_code: 'go',
      articles: [{ id: '9007199254740995', title: '内存模型', section_code: 'lifecycle' }] },
    { id: '5', code: 'empty', title: '正在整理的章节', module_code: 'go' },
  ],
})
beforeEach(() => {
  vi.useFakeTimers()
  vi.resetAllMocks()
  Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'visible' })
  Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', { configurable: true, value: scroll })
  detail.mockImplementation((code: string) => Promise.resolve(code === 'go' ? directory() : makeModule(code)))
  summaries.mockResolvedValue([directory()])
})
afterEach(() => { wrapper?.unmount(); wrapper = undefined; vi.useRealTimers(); vi.restoreAllMocks() })

async function openTopic(path = '/topics/go') {
  const router = createRouter({ history: createMemoryHistory(), routes: [
    { path: '/', component: { template: '<div />' } },
    { path: '/topics/:module', name: 'TopicOverview', component: TopicOverview },
    { path: '/blog/:module', name: 'BlogModule', component: { template: '<div />' } },
    { path: '/blog/:module/article/:article', name: 'BlogArticle', component: { template: '<div />' } },
  ] })
  await router.push(path)
  wrapper = mount({ template: '<router-view />' }, { attachTo: document.body, global: { plugins: [createPinia(), router] } })
  await flushPromises()
  return router
}

describe('public topic overview', () => {
  it('shows actual public chapters and exact article links in reading order without embedding or loading an article', async () => {
    const router = await openTopic()
    expect(wrapper!.get('h1').text()).toBe('Golang')
    expect(wrapper!.get('.topic-stats').text()).toContain('3 个章节，3 篇文章')
    expect(wrapper!.findAll('.article-row').map(row => row.attributes('href'))).toEqual([
      '/blog/go/article/9007199254740993', '/blog/go/article/9007199254740994', '/blog/go/article/9007199254740995',
    ])
    expect(wrapper!.text()).toContain('正在整理的章节')
    expect(wrapper!.find('iframe').exists()).toBe(false)
    expect(article).not.toHaveBeenCalled()
    expect(router.currentRoute.value.path).toBe('/topics/go')
    await wrapper!.get('.from-first').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/blog/go')
  })

  it('searches only current-topic titles, shows chapter ownership, and restores the catalog after clearing', async () => {
    await openTopic()
    await wrapper!.get('input[type="search"]').setValue('  INTERFACE  ')
    expect(wrapper!.findAll('.article-row')).toHaveLength(1)
    expect(wrapper!.get('.article-row').text()).toContain('Go Interface')
    expect(wrapper!.get('.chapter-section').text()).toContain('Go 语言基础')
    expect(wrapper!.get('.chapter-section').text()).toContain('类型系统')
    expect(wrapper!.get('.search-count').text()).toContain('1 篇')
    await wrapper!.get('input[type="search"]').setValue('没有这个标题')
    expect(wrapper!.findAll('.article-row')).toHaveLength(0)
    expect(wrapper!.get('.no-results').text()).toContain('没有找到匹配的文章标题')
    await wrapper!.get('.clear-search').trigger('click')
    expect(wrapper!.findAll('.article-row')).toHaveLength(3)
    expect(wrapper!.get('input').element).toHaveProperty('value', '')
  })

  it('uses chapter query for location and highlight without reloading or losing other query fields', async () => {
    const router = await openTopic('/topics/go?chapter=lifecycle&view=all#retained')
    expect(wrapper!.get('[aria-current="location"]').text()).toBe('Go 生命周期')
    expect(scroll).toHaveBeenCalled()
    expect(detail).toHaveBeenCalledTimes(1)
    await wrapper!.findAll('.chapter-index a')[0].trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({ chapter: 'basics', view: 'all' })
    expect(router.currentRoute.value.hash).toBe('#retained')
    expect(detail).toHaveBeenCalledTimes(1)
  })

  it('falls back to an available chapter for an unknown query and resets title search on theme change', async () => {
    const router = await openTopic('/topics/go?chapter=unknown')
    expect(wrapper!.get('[aria-current="location"]').text()).toBe('Go 语言基础')
    expect(router.currentRoute.value.query.chapter).toBe('unknown')
    expect(wrapper!.findAll('.article-row')).toHaveLength(3)
    await wrapper!.get('input').setValue('Interface')
    await router.push('/topics/ddd')
    await flushPromises()
    expect(wrapper!.get('input').element).toHaveProperty('value', '')
    expect(wrapper!.get('h1').text()).toBe('ddd')
    expect(wrapper!.findAll('.article-row')).toHaveLength(1)
  })

  it('shows an empty theme rather than first-article navigation and permits refreshing its directory', async () => {
    detail.mockResolvedValue(makeModule('go', []))
    const router = await openTopic()
    expect(wrapper!.text()).toContain('暂无已发布文章')
    expect(wrapper!.find('.from-first').exists()).toBe(false)
    expect(router.currentRoute.value.path).toBe('/topics/go')
    detail.mockResolvedValueOnce(directory())
    const retry = wrapper!.findAll('button').find(button => button.text() === '刷新目录')!
    await retry.trigger('click')
    await flushPromises()
    expect(wrapper!.findAll('.article-row')).toHaveLength(3)
  })

  it('keeps search results after a temporary refresh error and removes them after authoritative 404', async () => {
    await openTopic()
    await wrapper!.get('input').setValue('Interface')
    detail.mockRejectedValueOnce(new Error('offline'))
    await vi.advanceTimersByTimeAsync(60000)
    await flushPromises()
    expect(wrapper!.get('.refresh-warning').text()).toContain('上次成功读取')
    expect(wrapper!.findAll('.article-row')).toHaveLength(1)
    detail.mockRejectedValueOnce(new ApiError('gone', 404))
    await wrapper!.get('.refresh-warning button').trigger('click')
    await flushPromises()
    expect(wrapper!.text()).toContain('主题不可用')
    expect(wrapper!.findAll('.article-row')).toHaveLength(0)
    expect(wrapper!.find('iframe').exists()).toBe(false)
  })
})
