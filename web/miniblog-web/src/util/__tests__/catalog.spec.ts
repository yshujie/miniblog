import { describe, expect, it } from 'vitest'
import { mapModuleDetail } from '@/api/reading-contract'
import { firstArticle } from '@/util/reading'
import { catalogNeighbors, filterCatalog, findCatalogEntry, flattenCatalog } from '../catalog'

const mixedModule = () => mapModuleDetail({
  id: '1', code: 'go', title: 'Golang', sections: [
    { id: '2', code: 'empty', title: '空章节', module_code: 'go' },
    { id: '3', code: 'basics', title: '语言基础', module_code: 'go',
      articles: [{ id: '9007199254740995', title: '直属文章' }],
      subsections: [
        { id: '4', code: 'types', title: '类型系统', section_code: 'basics',
          articles: [{ id: '9007199254740993', title: '  Go Interface  ' }] },
        { id: '5', code: 'memory', title: '内存', section_code: 'basics',
          articles: [{ id: '9007199254740994', title: 'Go 内存模型' }] },
      ] },
    { id: '6', code: 'later', title: '后续章节', module_code: 'go',
      articles: [{ id: '9007199254740996', title: '最后一篇' }] },
  ],
})

describe('public directory derivation', () => {
  it('keeps backend chapter order, subsections before direct articles, and exact decimal IDs', () => {
    const module = mixedModule()
    const entries = flattenCatalog(module)
    expect(entries.map(entry => entry.article.id)).toEqual([
      '9007199254740993', '9007199254740994', '9007199254740995', '9007199254740996',
    ])
    expect(entries[0].article).toBe(firstArticle(module))
    expect(entries.every(entry => typeof entry.article.id === 'string')).toBe(true)
  })

  it('retains the original article and chapter references for breadcrumbs without altering the directory', () => {
    const module = mixedModule()
    const snapshot = JSON.stringify(module)
    const entries = flattenCatalog(module)
    const nested = findCatalogEntry(entries, '9007199254740993')!
    expect(nested.section).toBe(module.sections[1])
    expect(nested.subsection).toBe(module.sections[1].subsections[0])
    expect(nested.article).toBe(module.sections[1].subsections[0].articles[0])
    const direct = findCatalogEntry(entries, '9007199254740995')!
    expect(direct.section).toBe(module.sections[1])
    expect(direct.subsection).toBeUndefined()
    expect(JSON.stringify(module)).toBe(snapshot)
  })

  it('finds neighbors across subsection and chapter boundaries and handles the first and last articles', () => {
    const entries = flattenCatalog(mixedModule())
    expect(catalogNeighbors(entries, '9007199254740993')).toEqual({ previous: undefined, next: entries[1] })
    expect(catalogNeighbors(entries, '9007199254740995')).toEqual({ previous: entries[1], next: entries[3] })
    expect(catalogNeighbors(entries, '9007199254740996')).toEqual({ previous: entries[2], next: undefined })
  })

  it('does not invent a current article or first-article fallback for missing and empty placements', () => {
    const entries = flattenCatalog(mixedModule())
    expect(findCatalogEntry(entries, '404')).toBeUndefined()
    expect(catalogNeighbors(entries, '404')).toEqual({ previous: undefined, next: undefined })
    expect(flattenCatalog({ sections: [] })).toEqual([])
    expect(catalogNeighbors([], '1')).toEqual({ previous: undefined, next: undefined })
  })

  it('searches only titles, trims the query and ignores English case while preserving directory order', () => {
    const module = mixedModule()
    const entries = flattenCatalog(module)
    entries[2].article.tags = ['Interface']
    entries[2].article.content = 'Interface body text'
    const snapshot = JSON.stringify(module)
    expect(filterCatalog(entries, '  GO  ')).toEqual(entries.slice(0, 2))
    expect(filterCatalog(entries, '  INTERFACE  ')).toEqual([entries[0]])
    expect(filterCatalog(entries, '内存')).toEqual([entries[1]])
    expect(filterCatalog(entries, '  ')).toEqual(entries)
    expect(filterCatalog(entries, 'not-found')).toEqual([])
    expect(JSON.stringify(module)).toBe(snapshot)
  })
})
