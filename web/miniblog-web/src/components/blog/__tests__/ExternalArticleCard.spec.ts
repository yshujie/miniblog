import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { makeModule } from '@/__tests__/fixtures'
import { useUiStore } from '@/stores/ui'
import ExternalArticleCard from '../ExternalArticleCard.vue'
import { makeArticle } from '@/__tests__/fixtures'
let wrapper: VueWrapper | undefined
beforeEach(() => vi.useFakeTimers())
afterEach(() => { wrapper?.unmount(); vi.useRealTimers() })
describe('external article presentation', () => {
  it('prefers the current reading URL and does not recreate the frame for title changes', async () => {
    const article = makeArticle(); article.readingURL = 'https://example.com/current'
    wrapper = mount(ExternalArticleCard, { global: { plugins: [createPinia()] }, props: { article } })
    const frame = wrapper.get('iframe').element
    expect(wrapper.get('a').attributes('href')).toBe(article.readingURL)
    await wrapper.setProps({ article: { ...article, title: 'Changed title' } })
    expect(wrapper.get('iframe').element).toBe(frame)
    await wrapper.setProps({ article: { ...article, readingURL: 'javascript:bad' } })
    expect(wrapper.get('iframe').attributes('src')).toBe(article.externalLink)
  })
  it('keeps the original link available and offers only an advisory after ten seconds', async () => {
    wrapper = mount(ExternalArticleCard, { global: { plugins: [createPinia()] }, props: { article: makeArticle() } })
    expect(wrapper.get('a').attributes('href')).toBe('https://example.com/9007199254740993')
    expect(wrapper.get('a').attributes('rel')).toBe('noopener noreferrer')
    expect(wrapper.text()).toContain('正在打开文章')
    await vi.advanceTimersByTimeAsync(10000)
    expect(wrapper.text()).toContain('如无法显示，请打开原文')
    expect(wrapper.get('iframe').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('链接失效')
    await wrapper.get('iframe').trigger('load')
    expect(wrapper.find('.link-message').exists()).toBe(false)
    expect(wrapper.get('a').text()).toBe('打开原文')
  })
  it('does not embed or open a non HTTP(S) link', () => {
    const article = makeArticle()
    article.externalLink = 'javascript:alert(1)'
    wrapper = mount(ExternalArticleCard, { global: { plugins: [createPinia()] }, props: { article } })
    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(wrapper.find('a').exists()).toBe(false)
    expect(wrapper.text()).toContain('文章链接不可用')
  })
  it('resets the timer on changes and ignores an old iframe load event', async () => {
    wrapper = mount(ExternalArticleCard, { global: { plugins: [createPinia()] }, props: { article: makeArticle('1') } })
    const previousFrame = wrapper.get('iframe')
    await vi.advanceTimersByTimeAsync(9000)
    await wrapper.setProps({ article: makeArticle('2') })
    await previousFrame.trigger('load')
    await vi.advanceTimersByTimeAsync(1000)
    expect(wrapper.text()).toContain('正在打开文章')
    await vi.advanceTimersByTimeAsync(9000)
    expect(wrapper.text()).toContain('如无法显示')
    expect(wrapper.get('iframe').attributes('src')).toBe('https://example.com/2')
  })
})

describe('reading context and neighboring articles', () => {
  it('shows only provided metadata, uses the selected original URL, and routes neighbors with string IDs', async () => {
    const router = createRouter({ history: createMemoryHistory(), routes: [
      { path: '/topics/:module', name: 'TopicOverview', component: { template: '<div />' } },
      { path: '/blog/:module/article/:article', name: 'BlogArticle', component: { template: '<div />' } },
    ] })
    await router.push('/blog/go/article/9007199254740993')
    const module = makeModule('go', ['9007199254740993', '9007199254740994'])
    const article = makeArticle()
    article.author = 'Shujie'; article.tags = ['Go', '基础']; article.readingURL = 'https://example.com/current'
    const placement = { article: module.sections[0].articles[0], section: module.sections[0] }
    const next = { article: module.sections[0].articles[1], section: module.sections[0] }
    wrapper = mount(ExternalArticleCard, { global: { plugins: [createPinia(), router] }, props: { article, module, placement, next } })
    expect(wrapper.get('[aria-label="文章位置"]').text()).toContain('go')
    expect(wrapper.get('[aria-label="文章位置"]').text()).toContain('章')
    expect(wrapper.get('.reader-info').text()).toContain('Shujie')
    expect(wrapper.findAll('.reader-tag').map(tag => tag.text())).toEqual(['Go', '基础'])
    expect(wrapper.get('a[target="_blank"]').attributes('href')).toBe(article.readingURL)
    expect(wrapper.get('.reading-pagination .next').attributes('href')).toBe('/blog/go/article/9007199254740994')
    expect(wrapper.get('.reading-pagination .previous').text()).toContain('返回文章目录')
    const frame = wrapper.get('iframe').element
    await wrapper.setProps({ article: { ...article, author: '', tags: [], title: '新的完整文章标题' } })
    expect(wrapper.find('.reader-author').exists()).toBe(false)
    expect(wrapper.findAll('.reader-tag')).toHaveLength(0)
    expect(wrapper.get('iframe').element).toBe(frame)
  })
  it('hides pagination when the current article has no catalog placement', () => {
    wrapper = mount(ExternalArticleCard, { global: { plugins: [createPinia()] }, props: { article: makeArticle() } })
    expect(wrapper.find('.reading-pagination').exists()).toBe(false)
  })
})

describe('reading toolbar controls', () => {
  it('opens the directory panel and restores the desktop directory without touching the iframe', async () => {
    const pinia = createPinia()
    const ui = useUiStore(pinia)
    ui.sidebarOpen = false
    ui.openTopics()
    wrapper = mount(ExternalArticleCard, { global: { plugins: [pinia] }, props: { article: makeArticle() } })
    const frame = wrapper.get('iframe').element
    await wrapper.get('[aria-label="打开文章目录"]').trigger('click')
    expect(ui.mobileDrawerOpen).toBe(true)
    expect(ui.topicPickerOpen).toBe(false)
    expect(wrapper.get('[aria-label="打开文章目录"]').attributes('aria-expanded')).toBe('true')
    await wrapper.get('[aria-label="展开文章目录"]').trigger('click')
    expect(ui.sidebarOpen).toBe(true)
    expect(wrapper.get('iframe').element).toBe(frame)
  })
})
