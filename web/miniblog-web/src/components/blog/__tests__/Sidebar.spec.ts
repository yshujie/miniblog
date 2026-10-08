import { describe, expect, it } from 'vitest'
import { createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { mount, flushPromises } from '@vue/test-utils'
import Sidebar from '../Sidebar.vue'
import { useUiStore } from '@/stores/ui'
import { mapModuleDetail } from '@/api/reading-contract'
describe('shared desktop and drawer directory', () => {
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
