import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import Index from '../Index.vue'
import Blog from '../Blog.vue'
import SiteHeader from '@/components/SiteHeader.vue'
import { mapModuleDetail } from '@/api/reading-contract'
import { makeArticle } from '@/__tests__/fixtures'

const { modules, detail, article } = vi.hoisted(() => ({ modules: vi.fn(), detail: vi.fn(), article: vi.fn() }))
vi.mock('@/api/module', () => ({ fetchModules: modules }))
vi.mock('@/api/blog', () => ({ fetchModuleDetail: detail, fetchArticleDetail: article }))
beforeEach(() => {
  modules.mockReset(); detail.mockReset(); article.mockReset()
  const module = mapModuleDetail({
    id: '1', code: 'go', title: 'Go', sections: [
      { id: '2', code: 'empty', title: '空章节', module_code: 'go' },
      { id: '3', code: 's', title: '只有子章节', module_code: 'go',
        subsections: [{ id: '4', code: 'sub', section_code: 's', title: '子章',
          articles: [{ id: '9007199254740993', title: '首篇', section_code: 's', subsection_code: 'sub' }] }] },
    ],
  })
  modules.mockResolvedValue([module]); detail.mockResolvedValue(module)
  const selected = makeArticle(); selected.subsectionCode = 'sub'; article.mockResolvedValue(selected)
})
const stubs = {
  ElDrawer: { props: ['modelValue'], template: '<div v-if="modelValue" role="dialog"><slot /><slot name="header" /></div>' },
  ElButton: { template: '<button><slot /></button>' },
}
function makeRouter() {
  return createRouter({ history: createMemoryHistory(), routes: [
    { path: '/', component: { template: '<div />' } },
    { path: '/topics/:module', name: 'TopicOverview', component: { template: '<div>主题总览</div>' } },
    { path: '/blog/:module', name: 'BlogModule', component: Blog },
    { path: '/blog/:module/article/:article', name: 'BlogArticle', component: Blog },
  ] })
}
describe('topic navigation and preserved first article entry', () => {
  it.each(['/', '/blog/go/article/9007199254740993'])('header on %s enters overview without loading article detail', async (path) => {
    const router = makeRouter(); await router.push(path)
    const wrapper = mount(SiteHeader, { global: { plugins: [createPinia(), router], stubs } })
    await flushPromises()
    await wrapper.get('.site-nav a[href="/topics/go"]').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/topics/go')
    expect(article).not.toHaveBeenCalled()
    expect(detail).not.toHaveBeenCalled()
    wrapper.unmount()
  })
  it('home start button still reaches the first subsection article', async () => {
    const router = makeRouter(); await router.push('/')
    const wrapper = mount({ components: { Index }, template: '<Index /><router-view />' }, { global: { plugins: [createPinia(), router], stubs } })
    await flushPromises(); await wrapper.get('.read-btn').trigger('click'); await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/blog/go/article/9007199254740993')
    expect(article).toHaveBeenCalledTimes(1)
    expect(wrapper.find('iframe').exists()).toBe(true)
    wrapper.unmount()
  })
  it('a bookmarked bare module link retains the same first article behavior', async () => {
    const router = makeRouter(); await router.push('/blog/go')
    const wrapper = mount({ template: '<router-view />' }, { global: { plugins: [createPinia(), router], stubs } })
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/blog/go/article/9007199254740993')
    expect(article).toHaveBeenCalledTimes(1)
    expect(wrapper.find('iframe').exists()).toBe(true)
    wrapper.unmount()
  })
})
