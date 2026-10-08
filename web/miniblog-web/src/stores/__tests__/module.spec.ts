import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useModuleStore } from '../module'
import { makeModule, deferred } from '@/__tests__/fixtures'
import type { Module } from '@/types/module'

const { summaries, detail } = vi.hoisted(() => ({ summaries: vi.fn(), detail: vi.fn() }))
vi.mock('@/api/module', () => ({ fetchModules: summaries }))
vi.mock('@/api/blog', () => ({ fetchModuleDetail: detail }))
beforeEach(() => { setActivePinia(createPinia()); summaries.mockReset(); detail.mockReset() })

describe('module directory cache', () => {
  it('shares inflight requests and does not confuse partial lists with full empty directories', async () => {
    const store = useModuleStore()
    const summary = makeModule('go', [])
    summaries.mockResolvedValue([summary])
    await Promise.all([store.loadModules(), store.loadModules()])
    expect(summaries).toHaveBeenCalledTimes(1)
    expect(store.directoryCache.go.completeness).toBe('partial')
    const pending = deferred<Module>()
    detail.mockReturnValue(pending.promise)
    const first = store.loadModuleDetail('go')
    const second = store.loadModuleDetail('go')
    expect(detail).toHaveBeenCalledTimes(1)
    pending.resolve(summary)
    await Promise.all([first, second])
    expect(store.directoryCache.go.completeness).toBe('full')
    await store.loadModuleDetail('go')
    expect(detail).toHaveBeenCalledTimes(1)
    await store.loadModuleDetail('go', true)
    expect(detail).toHaveBeenCalledTimes(2)
  })

  it('retries failed directory loads and preserves full data when the module list arrives later', async () => {
    const store = useModuleStore()
    detail.mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce(makeModule())
    await expect(store.loadModuleDetail('go')).rejects.toThrow()
    await store.loadModuleDetail('go')
    summaries.mockResolvedValue([makeModule('go', [])])
    await store.loadModules()
    expect(store.modules[0].sections[0].articles).toHaveLength(1)
  })

  it('treats a successfully empty list as loaded, and permits explicit refresh', async () => {
    const store = useModuleStore()
    summaries.mockResolvedValue([])
    await store.loadModules()
    await store.loadModules()
    expect(store.listStatus).toBe('empty')
    expect(summaries).toHaveBeenCalledTimes(1)
    await store.loadModules(true)
    expect(summaries).toHaveBeenCalledTimes(2)
  })
})
