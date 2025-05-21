import http from '@/util/http'
import { Module } from '../types/module'
import { Article } from '../types/article'

// fetchModuleDetail 获取模块详情
export async function fetchModuleDetail(moduleCode: string): Promise<Module> {
  console.log('fetchModuleDetail', moduleCode)
  const { payload } = await http.get<{ module_detail: any }>(`/blog/moduleDetail?module_code=${moduleCode}`)
  console.log('payload', payload)
  return new Module(payload.module_detail)
}

// fetchArticleDetail 获取文章详情
export async function fetchArticleDetail(articleID: number): Promise<Article> {
  console.log('fetchArticleDetail', articleID)
  const { payload } = await http.get<{ article_detail: any }>(`/blog/articleDetail?article_id=${articleID}`)
  console.log('payload', payload)
  return new Article(payload.article_detail)
}