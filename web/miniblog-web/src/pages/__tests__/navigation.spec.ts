import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import Index from '../Index.vue'
import Blog from '../Blog.vue'
import Header from '@/components/Header.vue'
import BlogHeader from '@/components/blog/BlogHeader.vue'
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
  modules.mockResolvedValue([module])
  detail.mockResolvedValue(module)
  const selected = makeArticle()
  selected.subsectionCode = 'sub'
  article.mockResolvedValue(selected)
})

describe('shared first article navigation', () => {
  it.each([
    ['home start button', Index, '.read-btn'],
    ['home header module', Header, 'button.nav-link'],
    ['reading header module', BlogHeader, '.blog-nav-item'],
  ])('%s reaches the same subsection article through the route coordinator', async (_name, Entry, selector) => {
    const router = createRouter({ history: createMemoryHistory(), routes: [
      { path: '/', component: { template: '<div />' } },
      { path: '/blog/:module', name: 'BlogModule', component: Blog },
      { path: '/blog/:module/article/:article', name: 'BlogArticle', component: Blog },
    ] })
    await router.push('/')
    const wrapper = mount({ components: { Entry }, template: '<Entry /><router-view />' }, {
      global: { plugins: [createPinia(), router], stubs: {
        ElDrawer: { props: ['modelValue'], template: '<div v-if="modelValue"><slot /></div>' },
        ElButton: { template: '<button><slot /></button>' },
        ElAvatar: { template: '<div><slot /></div>' },
      } },
    })
    await flushPromises()
    await wrapper.get(selector as string).trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/blog/go/article/9007199254740993')
    expect(detail).toHaveBeenCalledTimes(1)
    expect(article).toHaveBeenCalledTimes(1)
    expect(wrapper.find('iframe').exists()).toBe(true)
    wrapper.unmount()
  })
})
