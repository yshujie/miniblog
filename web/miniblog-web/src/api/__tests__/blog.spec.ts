import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fetchArticleDetail, fetchModuleDetail } from '../blog'
import { Article } from '@/types/article'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/util/http', () => ({ default: { get } }))

beforeEach(() => get.mockReset())

describe('public v1 reading contract', () => {
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
