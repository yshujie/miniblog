import { mapModuleDetail } from '@/api/reading-contract'
import { Article } from '@/types/article'

export const makeArticle = (id = '9007199254740993', moduleCode = 'go') => new Article({
  id, moduleCode, sectionCode: 's', title: '文章 ' + id, externalLink: 'https://example.com/' + id,
})
export const makeModule = (code = 'go', ids = ['9007199254740993']) => mapModuleDetail({
  id: '1', code, title: code, sections: [{
    id: '2', code: 's', title: '章', module_code: code,
    articles: ids.map(id => ({ id, title: '文章 ' + id, section_code: 's', module_code: code })),
  }],
})
export function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => { resolve = res; reject = rej })
  return { promise, resolve, reject }
}
