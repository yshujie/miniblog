import { describe, expect, it } from 'vitest'
import { mapModuleDetail, decimalID } from '@/api/reading-contract'
import { firstArticle, safeExternalURL, articleTitle, articleReadingLinks } from '../reading'
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


describe('Notion original and embed addresses', () => {
  const pageID = '2ec330bdddf1802d9835c5eba0858758'
  const origin = 'https://shujie-blog.notion.site'
  const current = `${origin}/defer-panic-recover-${pageID}?pvs=4#part`
  const embed = `${origin}/ebd//${pageID}?showTitle=true`
  const links = (readingURL: string, externalLink = '') => articleReadingLinks({ readingURL, externalLink })

  it('keeps the current public original but derives an embed independently of the legacy URL', () => {
    expect(links(current, embed)).toEqual({ originalURL: current, embedURL: `${origin}/ebd/${pageID}` })
  })

  it.each(['', 'javascript:bad', `${origin}/ebd/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`, 'https://old-domain.notion.site/ebd/' + pageID])('derives an embed from the current page instead of a missing, unsafe or stale legacy link: %s', legacy => {
    expect(links(current, legacy)).toEqual({ originalURL: current, embedURL: `${origin}/ebd/${pageID}` })
  })

  it('keeps an existing current embed authoritative over a different legacy page', () => {
    expect(links(embed, `${origin}/ebd/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`)).toEqual({ originalURL: embed, embedURL: embed })
  })

  it('derives the current page identity independently of slug and hexadecimal case', () => {
    expect(links(`${origin}/new-slug-${pageID.toUpperCase()}`, embed).embedURL).toBe(`${origin}/ebd/${pageID}`)
  })

  it('converts a legacy ordinary Notion page when there is no safe current URL', () => {
    expect(links('javascript:bad', current)).toEqual({ originalURL: current, embedURL: `${origin}/ebd/${pageID}` })
  })

  it.each(['https://example.com/doc', 'https://notion.site.example.com/' + pageID, `${origin}/named-page-without-id`, `${origin}/${pageID}?p=`, `${origin}/${pageID}?v=other`, `https://app.notion.com/p/${pageID}`])('does not guess a Notion page or overwrite a current unrelated address: %s', address => {
    expect(links(address, embed)).toEqual({ originalURL: address, embedURL: address })
  })

  it.each([`${origin}/${pageID}/`, `${origin}/named-2ec330bd-ddf1-802d-9835-c5eba0858758`, `https://www.notion.so/${pageID}`])('handles explicit page IDs and keeps the current public origin: %s', address => {
    expect(links(address).originalURL).toBe(address)
    expect(links(address).embedURL).toBe(new URL(address).origin + '/ebd/' + pageID)
  })

  it('does not create an iframe or original link from unsafe input', () => {
    expect(links('data:text/html,blocked', 'https://user:password@shujie-blog.notion.site/' + pageID)).toEqual({ originalURL: null, embedURL: null })
  })
})
