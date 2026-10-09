import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { useHomeTopics } from '../useHomeTopics'
import { useModuleStore } from '@/stores/module'
import { Module } from '@/types/module'
import { deferred, makeModule } from '@/__tests__/fixtures'

const { summaries, detail } = vi.hoisted(() => ({ summaries: vi.fn(), detail: vi.fn() }))
vi.mock('@/api/module', () => ({ fetchModules: summaries }))
vi.mock('@/api/blog', () => ({ fetchModuleDetail: detail }))

let intersection: IntersectionObserverCallback
const observed = new Set<Element>()
beforeEach(() => {
  summaries.mockReset(); detail.mockReset(); observed.clear()
  vi.stubGlobal('IntersectionObserver', class {
    constructor(callback: IntersectionObserverCallback) { intersection = callback }
    observe(element: Element) { observed.add(element) }
    unobserve(element: Element) { observed.delete(element) }
    disconnect() { observed.clear() }
  })
})
afterEach(() => { vi.unstubAllGlobals(); vi.useRealTimers() })
const summary = (code: string) => new Module({ id: '1', code, title: code })
async function mountHome(pinia = createPinia()) {
  let home!: ReturnType<typeof useHomeTopics>
  const wrapper = mount(defineComponent({
    setup() { home = useHomeTopics(); return home },
    template: '<section v-for="topic in store.modules" :key="topic.code" :data-topic="topic.code" :ref="element => registerTopic(element, topic.code)">{{ topic.title }}</section>',
  }), { global: { plugins: [pinia] } })
  await flushPromises()
  const visible = (code: string) => {
    const element = wrapper.get('[data-topic="' + code + '"]').element
    intersection([{ target: element, isIntersecting: true } as IntersectionObserverEntry], {} as IntersectionObserver)
  }
  return { wrapper, home, visible }
}
describe('home topic previews', () => {
  it('shows summaries immediately and loads details only near the viewport', async () => {
    summaries.mockResolvedValue([summary('go'), summary('other')])
    const pending = deferred<Module>(); detail.mockReturnValue(pending.promise)
    const { wrapper, home, visible } = await mountHome()
    expect(wrapper.text()).toContain('other')
    expect(detail).not.toHaveBeenCalled()
    expect(home.previews.go?.module).toBeNull()
    visible('go'); await flushPromises()
    expect(detail).toHaveBeenCalledWith('go')
    expect(home.previews.go.status).toBe('loading')
    expect(home.previews.go.module).toBeNull()
    pending.resolve(makeModule('go')); await flushPromises()
    expect(home.previews.go.status).toBe('success')
    expect(home.previews.go.module?.sections[0].articles).toHaveLength(1)
    expect(detail).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })
  it('limits auxiliary detail loading to three concurrent requests', async () => {
    summaries.mockResolvedValue(['one', 'two', 'three', 'four'].map(summary))
    const pending = new Map<string, ReturnType<typeof deferred<Module>>>()
    detail.mockImplementation((code: string) => {
      const request = deferred<Module>(); pending.set(code, request); return request.promise
    })
    const { wrapper, visible } = await mountHome()
    for (const code of ['one', 'two', 'three', 'four']) visible(code)
    await flushPromises()
    expect(detail).toHaveBeenCalledTimes(3)
    pending.get('one')!.resolve(makeModule('one')); await flushPromises()
    expect(detail).toHaveBeenCalledTimes(4)
    expect(detail).toHaveBeenLastCalledWith('four')
    wrapper.unmount()
    for (const [code, request] of pending) request.resolve(makeModule(code))
    await flushPromises()
  })
  it('isolates one preview failure and retries only that topic', async () => {
    summaries.mockResolvedValue([summary('go'), summary('other')])
    detail.mockImplementation((code: string) => code === 'go' ? Promise.reject(new Error('offline')) : Promise.resolve(makeModule(code)))
    const { wrapper, home, visible } = await mountHome()
    visible('go'); visible('other'); await flushPromises()
    expect(home.previews.go.status).toBe('error')
    expect(home.previews.other.status).toBe('success')
    expect(wrapper.text()).toContain('go')
    detail.mockResolvedValue(makeModule('go'))
    home.retryTopic('go'); await flushPromises()
    expect(home.previews.go.status).toBe('success')
    expect(detail.mock.calls.map(call => call[0])).toEqual(['go', 'other', 'go'])
    wrapper.unmount()
  })
  it('reuses a complete cached directory and never mistakes a summary for zero articles', async () => {
    const pinia = createPinia()
    summaries.mockResolvedValue([summary('go')]); detail.mockResolvedValue(makeModule('go'))
    await useModuleStore(pinia).loadModuleDetail('go')
    const { wrapper, home, visible } = await mountHome(pinia)
    visible('go'); await flushPromises()
    expect(home.previews.go.status).toBe('success')
    expect(home.previews.go.module?.sections).toHaveLength(1)
    expect(detail).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })
  it('refreshes an expired complete cache only when the topic enters view', async () => {
    vi.useFakeTimers(); vi.setSystemTime(0)
    const pinia = createPinia()
    summaries.mockResolvedValue([summary('go')]); detail.mockResolvedValue(makeModule('go'))
    await useModuleStore(pinia).loadModuleDetail('go')
    vi.setSystemTime(60000)
    const { wrapper, home, visible } = await mountHome(pinia)
    expect(detail).toHaveBeenCalledTimes(1)
    const pending = deferred<Module>(); detail.mockReturnValue(pending.promise)
    visible('go'); await flushPromises()
    expect(detail).toHaveBeenCalledTimes(2)
    expect(home.previews.go.status).toBe('loading')
    expect(home.previews.go.module).toBeNull()
    pending.resolve(makeModule('go', ['1', '2'])); await flushPromises()
    expect(home.previews.go.status).toBe('success')
    expect(home.previews.go.module?.sections[0].articles).toHaveLength(2)
    wrapper.unmount()
  })
  it('loads a removed topic again when the same code returns to the viewport', async () => {
    summaries.mockResolvedValueOnce([summary('go')]).mockResolvedValueOnce([]).mockResolvedValueOnce([summary('go')])
    detail.mockResolvedValue(makeModule('go'))
    const { wrapper, home, visible } = await mountHome()
    visible('go'); await flushPromises()
    expect(detail).toHaveBeenCalledTimes(1)
    await home.store.loadModules(true); await flushPromises()
    expect(home.previews.go).toBeUndefined()
    await home.store.loadModules(true); await flushPromises()
    expect(home.previews.go.status).toBe('idle')
    visible('go'); await flushPromises()
    expect(detail).toHaveBeenCalledTimes(2)
    expect(home.previews.go.status).toBe('success')
    wrapper.unmount()
  })
  it('disconnects observation and ignores a late detail result after unmount', async () => {
    summaries.mockResolvedValue([summary('go')])
    const pending = deferred<Module>(); detail.mockReturnValue(pending.promise)
    const { wrapper, home, visible } = await mountHome()
    visible('go'); await flushPromises()
    wrapper.unmount()
    const before = JSON.stringify(home.previews)
    pending.resolve(makeModule('go')); await flushPromises()
    expect(JSON.stringify(home.previews)).toBe(before)
    expect(observed.size).toBe(0)
  })
})
