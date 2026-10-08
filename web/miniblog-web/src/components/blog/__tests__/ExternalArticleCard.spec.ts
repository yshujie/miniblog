import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import ExternalArticleCard from '../ExternalArticleCard.vue'
import { makeArticle } from '@/__tests__/fixtures'
let wrapper: VueWrapper | undefined
beforeEach(() => vi.useFakeTimers())
afterEach(() => { wrapper?.unmount(); vi.useRealTimers() })
describe('external article presentation', () => {
  it('keeps the original link available and offers only an advisory after ten seconds', async () => {
    wrapper = mount(ExternalArticleCard, { props: { article: makeArticle() } })
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
    wrapper = mount(ExternalArticleCard, { props: { article } })
    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(wrapper.find('a').exists()).toBe(false)
    expect(wrapper.text()).toContain('文章链接不可用')
  })
  it('resets the timer on changes and ignores an old iframe load event', async () => {
    wrapper = mount(ExternalArticleCard, { props: { article: makeArticle('1') } })
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
