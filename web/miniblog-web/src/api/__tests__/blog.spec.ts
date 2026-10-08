import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fetchArticleDetail, fetchModuleDetail } from '../blog'
import { fetchModules } from '../module'
import { Article } from '@/types/article'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/util/http', () => ({ default: { get } }))

beforeEach(() => get.mockReset())

describe('public v1 reading contract', () => {
  it('accepts the legacy module-list payload with zero summary IDs', async () => {
    get.mockResolvedValue({ payload: { modules: [
      { id: 0, code: 'go', title: 'Go', status: 0, sort: 1 },
      { id: '0', code: 'java', title: 'Java', status: 0, sort: 2 },
    ] } })
    const modules = await fetchModules()
    expect(get).toHaveBeenCalledWith('/blog/modules')
    expect(modules.map(module => ({ id: module.id, code: module.code, title: module.title }))).toEqual([
      { id: '', code: 'go', title: 'Go' },
      { id: '', code: 'java', title: 'Java' },
    ])
    expect(modules[0].sections).toEqual([])
  })

  it('locates summaries without IDs by code and preserves provided exact IDs', async () => {
    get.mockResolvedValue({ payload: { modules: [
      { code: 'missing', title: 'Missing ID' },
      { id: null, code: 'null', title: 'Null ID' },
      { id: '', code: 'empty', title: 'Empty ID' },
      { id: 1, code: 'normal', title: 'Normal ID' },
      { id: '9007199254740993', code: 'large', title: 'Large ID' },
    ] } })
    expect((await fetchModules()).map(module => module.id)).toEqual(['', '', '', '1', '9007199254740993'])
  })

  it('still rejects a zero article ID from an article response', async () => {
    get.mockResolvedValue({ payload: { article_detail: { id: '0', title: 'Invalid article' } } })
    await expect(fetchArticleDetail('0')).rejects.toThrow('文章 ID 无效')
  })

  it('preserves a decimal ID beyond JavaScript number precision', async () => {
    get.mockResolvedValue({ payload: { article_detail: {
      id: '9007199254740993', title: '文章', module_code: 'new', section_code: 's', external_link: 'https://example.com/a',
    } } })
    const article = await fetchArticleDetail('9007199254740993')
    expect(article).toBeInstanceOf(Article)
    expect(article.id).toBe('9007199254740993')
    expect(article.moduleCode).toBe('new')
    expect(article.externalLink).toBe('https://example.com/a')
    expect(article.tags).toEqual([])
  })

  it('maps both direct and subsection articles and accepts null lists', async () => {
    get.mockResolvedValue({ payload: { module_detail: {
      id: '1', code: 'go', title: 'Go', sections: [
        { id: '2', code: 's', module_code: 'go', title: '章', articles: [{ id: '3', title: '直属' }],
          subsections: [{ id: '4', code: 'sub', section_code: 's', title: '子章', articles: [{ id: '5', title: '子文' }] }] },
        { id: '6', code: 'empty', title: '空章', articles: null, subsections: null },
      ],
    } } })
    const module = await fetchModuleDetail('go')
    expect(module.sections[0].articles[0].id).toBe('3')
    expect(module.sections[0].subsections[0].articles[0].id).toBe('5')
    expect(module.sections[1].articles).toEqual([])
    expect(module.sections[1].subsections).toEqual([])
  })
})
