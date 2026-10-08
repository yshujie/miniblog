import { describe, expect, it } from 'vitest'
import { mapModuleDetail, decimalID } from '@/api/reading-contract'
import { firstArticle, safeExternalURL, articleTitle } from '../reading'
import { parseResponse } from '../http'
import { makeArticle } from '@/__tests__/fixtures'

describe('reading selection and v1 IDs', () => {
  it('skips empty chapters and chooses subsection articles before direct articles', () => {
    const module = mapModuleDetail({
      id: '1', code: 'go', title: 'Go', sections: [
        { id: '2', code: 'empty', module_code: 'go', title: '空章' },
        { id: '3', code: 's', module_code: 'go', title: '章',
          articles: [{ id: '4', title: '直属' }],
          subsections: [{ id: '5', code: 'sub', title: '子章', section_code: 's',
            articles: [{ id: '6', title: '首篇' }] }] },
      ],
    })
    expect(firstArticle(module)?.id).toBe('6')
    expect(firstArticle(mapModuleDetail({ id: '1', code: 'empty', title: '空' }))).toBeUndefined()
  })

  it('preserves numeric IDs before JSON conversion without changing unrelated numbers', () => {
    const value = parseResponse('{"payload":{"id" : 9007199254740993,"pos":10,"child":{"id":9223372036854775807}}}')
    expect(value).toEqual({ payload: { id: '9007199254740993', pos: 10, child: { id: '9223372036854775807' } } })
    expect(decimalID('9007199254740993')).toBe('9007199254740993')
    expect(() => decimalID(Number('9007199254740993'))).toThrow()
  })

  it.each(['0', '-1', '1e3', 'NaN', '9223372036854775808', '123456789012345678901'])('rejects invalid or out-of-range IDs: %s', id => {
    expect(() => decimalID(id)).toThrow()
  })

  it.each(['javascript:alert(1)', 'data:text/html,x', '/relative', 'https://user:password@example.com'])('blocks unsuitable external URLs: %s', value => {
    expect(safeExternalURL(value)).toBeNull()
  })

  it('provides HTTP(S) original links and a stable title fallback', () => {
    expect(safeExternalURL('https://example.com/doc#part')).toBe('https://example.com/doc#part')
    expect(safeExternalURL('http://example.com/doc')).toBe('http://example.com/doc')
    const article = makeArticle()
    article.title = ''
    expect(articleTitle(article)).toBe('example.com')
  })
})
