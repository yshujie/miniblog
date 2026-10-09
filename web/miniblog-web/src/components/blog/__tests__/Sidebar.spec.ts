import { describe, expect, it } from 'vitest'
import { createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { mount, flushPromises } from '@vue/test-utils'
import Sidebar from '../Sidebar.vue'
import { useUiStore } from '@/stores/ui'
import { mapModuleDetail } from '@/api/reading-contract'
import { firstArticle } from '@/util/reading'
describe('shared desktop and drawer directory', () => {
  it('keeps subsection articles before direct articles in a mixed directory', async () => {
    const pinia = createPinia()
    const router = createRouter({ history: createMemoryHistory(), routes: [
      { path: '/blog/:module/article/:article', name: 'BlogArticle', component: { template: '<div />' } },
    ] })
    const module = mapModuleDetail({ id: '1', code: 'go', title: 'Go', sections: [
      { id: '2', code: 's', title: '章节', module_code: 'go', articles: [{ id: '5', title: '直属' }],
        subsections: [{ id: '3', code: 'sub', title: '子章', section_code: 's',
          articles: [{ id: '9007199254740993', title: '子章首篇' }, { id: '4', title: '子章次篇' }] }] },
    ] })
    await router.push('/blog/go/article/9007199254740993')
    const wrapper = mount(Sidebar, { props: { sections: module.sections, moduleCode: 'go', drawer: true }, global: { plugins: [pinia, router] } })
    const buttons = wrapper.findAll('.article-item')
    expect(buttons.map(button => button.text())).toEqual(['子章首篇', '子章次篇', '直属'])
    expect(firstArticle(module)?.id).toBe('9007199254740993')
    await buttons[0].trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.params.article).toBe(firstArticle(module)?.id)
    wrapper.unmount()
  })
  it('uses accessible buttons, expands the current path and closes the drawer on selection', async () => {
    const pinia = createPinia()
    const router = createRouter({ history: createMemoryHistory(), routes: [
      { path: '/blog/:module/article/:article', name: 'BlogArticle', component: { template: '<div />' } },
    ] })
    await router.push('/blog/go/article/4')
    const module = mapModuleDetail({
      id: '1', code: 'go', title: 'Go', sections: [{
        id: '2', code: 's', title: '章节', module_code: 'go', articles: [{ id: '5', title: '直属' }],
        subsections: [{ id: '3', code: 'sub', title: '子章', section_code: 's', articles: [{ id: '4', title: '当前文章' }] }],
      }],
    })
    const ui = useUiStore(pinia)
    ui.sidebarOpen = false
    ui.openDrawer()
    const wrapper = mount(Sidebar, { props: { sections: module.sections, moduleCode: 'go', moduleTitle: 'Go', drawer: true },
      global: { plugins: [pinia, router] } })
    expect(wrapper.classes()).toContain('sidebar-drawer')
    expect(wrapper.attributes('inert')).toBeUndefined()
    expect(wrapper.get('.section-header').attributes('aria-expanded')).toBe('true')
    expect(wrapper.get('[aria-current="page"]').text()).toBe('当前文章')
    expect(wrapper.findAll('.article-item').map(button => button.text())).toEqual(['当前文章', '直属'])
    await wrapper.get('.section-header').trigger('click')
    expect(wrapper.get('.section-header').attributes('aria-expanded')).toBe('false')
    await wrapper.findAll('.article-item')[1].trigger('click')
    await flushPromises()
    expect(ui.mobileDrawerOpen).toBe(false)
    expect(router.currentRoute.value.params.article).toBe('5')
    wrapper.unmount()
  })
  it('makes a collapsed desktop directory inert', () => {
    const pinia = createPinia()
    const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/', component: { template: '<div />' } }] })
    useUiStore(pinia).sidebarOpen = false
    const wrapper = mount(Sidebar, { props: { sections: [], moduleCode: 'go' }, global: { plugins: [pinia, router] } })
    expect(wrapper.attributes('inert')).toBeDefined()
    wrapper.unmount()
  })
})

async function mountDirectory() {
  const pinia = createPinia()
  const router = createRouter({ history: createMemoryHistory(), routes: [
    { path: '/blog/:module/article/:article', name: 'BlogArticle', component: { template: '<div />' } },
    { path: '/topics/:module', name: 'TopicOverview', component: { template: '<div />' } },
  ] })
  await router.push('/blog/go/article/4')
  const module = mapModuleDetail({ id: '1', code: 'go', title: 'Go', sections: [
    { id: '2', code: 'current', title: '当前章节', module_code: 'go',
      subsections: [{ id: '3', code: 'sub', title: '当前子章', section_code: 'current', articles: [{ id: '4', title: '当前文章' }] }] },
    { id: '6', code: 'other', title: '其他章节', module_code: 'go', articles: [{ id: '7', title: 'Go concurrency：这是一个需要完整展示而不能提前截断的长文章标题' }] },
  ] })
  const wrapper = mount(Sidebar, { props: { sections: module.sections, moduleCode: 'go', moduleTitle: 'Go', drawer: true }, global: { plugins: [pinia, router] } })
  return { wrapper, router, module }
}
describe('directory exploration', () => {
  it('initially expands only the current path and preserves user choices across directory refreshes', async () => {
    const { wrapper, module } = await mountDirectory()
    const chapters = () => wrapper.findAll('.section-header')
    expect(chapters().map(button => button.attributes('aria-expanded'))).toEqual(['true', 'false'])
    await chapters()[1].trigger('click')
    await wrapper.setProps({ sections: module.sections.map(section => ({ ...section, title: section.title + '更新' })) })
    expect(chapters().map(button => button.attributes('aria-expanded'))).toEqual(['true', 'true'])
    await chapters()[0].trigger('click')
    await wrapper.setProps({ sections: [...module.sections] })
    expect(chapters()[0].attributes('aria-expanded')).toBe('true')
    await wrapper.setProps({ sections: [module.sections[0]] })
    await wrapper.setProps({ sections: module.sections })
    expect(chapters()[1].attributes('aria-expanded')).toBe('false')
    wrapper.unmount()
  })
  it('filters titles without shortening them, temporarily expands matches, and restores expansion on clearing', async () => {
    const { wrapper, module } = await mountDirectory()
    const search = wrapper.get('input[type="search"]')
    await search.setValue('  GO CONCURRENCY  ')
    expect(wrapper.findAll('.section-header')).toHaveLength(1)
    expect(wrapper.get('.section-header').attributes('aria-expanded')).toBe('true')
    expect(wrapper.get('.article-title').text()).toBe(module.sections[1].articles[0].title)
    expect(module.sections[0].subsections[0].articles).toHaveLength(1)
    await search.setValue('not a title')
    expect(wrapper.findAll('.article-item')).toHaveLength(0)
    expect(wrapper.get('[role="status"]').text()).toContain('没有匹配')
    await wrapper.get('button[aria-label="清空标题搜索"]').trigger('click')
    expect(wrapper.findAll('.section-header').map(button => button.attributes('aria-expanded'))).toEqual(['true', 'false'])
    expect(wrapper.get('[aria-current="page"]').text()).toBe('当前文章')
    wrapper.unmount()
  })
  it('expands a newly selected path and resets exploration when the module changes', async () => {
    const { wrapper, router } = await mountDirectory()
    await router.push('/blog/go/article/7')
    await flushPromises()
    expect(wrapper.findAll('.section-header')[1].attributes('aria-expanded')).toBe('true')
    await wrapper.get('input[type="search"]').setValue('concurrency')
    await wrapper.setProps({ moduleCode: 'new' })
    expect((wrapper.get('input[type="search"]').element as HTMLInputElement).value).toBe('')
    expect(wrapper.findAll('.section-header').map(button => button.attributes('aria-expanded'))).toEqual(['false', 'true'])
    wrapper.unmount()
  })
})
