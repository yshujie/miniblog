import { describe, expect, it, vi } from 'vitest'
import { createReaderLoader } from '../reader-loader'
import { ApiError } from '@/util/http'
import { deferred, makeArticle, makeModule } from '@/__tests__/fixtures'
import type { Article } from '@/types/article'
import type { Module } from '@/types/module'

function setup() {
  const loadArticle = vi.fn().mockResolvedValue(makeArticle())
  const loadModule = vi.fn().mockResolvedValue(makeModule())
  const replace = vi.fn().mockResolvedValue(undefined)
  return { loadArticle, loadModule, replace, reader: createReaderLoader({ loadArticle, loadModule, replace }) }
}

describe('single route reading coordinator', () => {
  it('resolves a moved article before the historical module and carries it across canonical replacement', async () => {
    const { reader, loadArticle, loadModule, replace } = setup()
    loadArticle.mockResolvedValue(makeArticle('9007199254740993', 'new'))
    loadModule.mockResolvedValue(makeModule('new'))
    await reader.load({ moduleCode: 'deleted-old', articleId: '9007199254740993', query: { from: 'bookmark' }, hash: '#part' })
    expect(loadModule).not.toHaveBeenCalled()
    expect(replace).toHaveBeenCalledWith({
      name: 'BlogArticle', params: { module: 'new', article: '9007199254740993' }, query: { from: 'bookmark' }, hash: '#part',
    })
    await reader.load({ moduleCode: 'new', articleId: '9007199254740993' })
    expect(loadArticle).toHaveBeenCalledTimes(1)
    expect(loadModule).toHaveBeenCalledWith('new', false)
    expect(reader.state.status).toBe('success')
    expect(reader.state.article?.id).toBe('9007199254740993')
  })

  it('never replaces a missing or unpublished requested article with a first article', async () => {
    const { reader, loadArticle, loadModule, replace } = setup()
    loadArticle.mockRejectedValue(new ApiError('missing', 404))
    await reader.load({ moduleCode: 'go', articleId: '99' })
    expect(reader.state.status).toBe('not_found')
    expect(loadModule).not.toHaveBeenCalled()
    expect(replace).not.toHaveBeenCalled()
  })

  it('identifies a missing current directory separately from a missing article', async () => {
    const { reader, loadModule, replace } = setup()
    loadModule.mockRejectedValue(new ApiError('missing directory', 404))
    await reader.load({ moduleCode: 'go', articleId: '9007199254740993' })
    expect(reader.state.status).toBe('not_found')
    expect(reader.state.resource).toBe('module')
    expect(reader.state.message).toBe('模块不存在或不可用')
    expect(replace).not.toHaveBeenCalled()
  })

  it('cancels prior article requests and prevents late success/finally from affecting a newer request', async () => {
    const { reader, loadArticle } = setup()
    const a = deferred<Article>()
    const b = deferred<Article>()
    loadArticle.mockReturnValueOnce(a.promise).mockReturnValueOnce(b.promise)
    const first = reader.load({ moduleCode: 'go', articleId: '1' })
    const second = reader.load({ moduleCode: 'go', articleId: '9007199254740993' })
    expect(loadArticle.mock.calls[0][1].aborted).toBe(true)
    a.resolve(makeArticle('1'))
    await first
    expect(reader.state.status).toBe('loading')
    expect(reader.state.busy).toBe(true)
    expect(reader.state.article).toBeNull()
    b.resolve(makeArticle())
    await second
    expect(reader.state.status).toBe('success')
    expect(reader.state.busy).toBe(false)
  })

  it('ignores an obsolete request failure while a later navigation succeeds', async () => {
    const { reader, loadArticle } = setup()
    const old = deferred<Article>()
    loadArticle.mockReturnValueOnce(old.promise).mockResolvedValueOnce(makeArticle())
    const first = reader.load({ moduleCode: 'go', articleId: '1' })
    await reader.load({ moduleCode: 'go', articleId: '9007199254740993' })
    old.reject(new Error('old failure'))
    await first
    expect(reader.state.status).toBe('success')
    expect(reader.state.article?.id).toBe('9007199254740993')
  })

  it('stops waiting for shared directories without letting their result overwrite the current route', async () => {
    const { reader, loadModule } = setup()
    const old = deferred<Module>()
    loadModule.mockReturnValueOnce(old.promise).mockResolvedValueOnce(makeModule('empty', []))
    const first = reader.load({ moduleCode: 'old' })
    await reader.load({ moduleCode: 'empty' })
    await first
    old.resolve(makeModule('old'))
    expect(reader.state.module?.code).toBe('empty')
    expect(reader.state.status).toBe('empty')
    expect(reader.state.busy).toBe(false)
  })

  it('handles empty modules and explicit retry without mistaking errors for absence', async () => {
    const { reader, loadModule, replace } = setup()
    loadModule.mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce(makeModule('go', []))
    await reader.load({ moduleCode: 'go' })
    expect(reader.state.status).toBe('error')
    await reader.load({ moduleCode: 'go' }, true)
    expect(loadModule).toHaveBeenLastCalledWith('go', true)
    expect(reader.state.status).toBe('empty')
    expect(replace).not.toHaveBeenCalled()
  })

  it('redirects bare module links to their first article and rejects malformed article IDs locally', async () => {
    const { reader, replace, loadArticle } = setup()
    await reader.load({ moduleCode: 'go' })
    expect(replace).toHaveBeenCalledWith(expect.objectContaining({
      params: { module: 'go', article: '9007199254740993' },
    }))
    await reader.load({ moduleCode: 'go', articleId: '-1' })
    expect(reader.state.status).toBe('not_found')
    expect(loadArticle).not.toHaveBeenCalled()
  })

  it('refreshes an outdated directory after the article changed position', async () => {
    const { reader, loadModule } = setup()
    loadModule.mockResolvedValueOnce(makeModule('go', [])).mockResolvedValueOnce(makeModule())
    await reader.load({ moduleCode: 'go', articleId: '9007199254740993' })
    expect(loadModule.mock.calls.map(call => call[1])).toEqual([false, true])
    expect(reader.state.status).toBe('success')
  })
})
