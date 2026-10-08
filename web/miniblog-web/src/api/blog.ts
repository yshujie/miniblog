import http from '@/util/http'
import { mapArticle, mapModuleDetail, type ArticleDTO, type ModuleDetailDTO } from './reading-contract'

export async function fetchModuleDetail(moduleCode: string) {
  const { payload } = await http.get<{ module_detail: ModuleDetailDTO }>('/blog/moduleDetail', {
    params: { module_code: moduleCode },
  })
  return mapModuleDetail(payload.module_detail)
}

export async function fetchArticleDetail(articleID: string, signal?: AbortSignal) {
  const { payload } = await http.get<{ article_detail: ArticleDTO }>('/blog/articleDetail', {
    params: { article_id: articleID }, signal,
  })
  return mapArticle(payload.article_detail)
}
