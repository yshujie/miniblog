import http from '@/util/http'
import { Module } from '../types/module'
import { Article } from '../types/article'
import { Section } from '../types/section'
import { Subsection } from '../types/subsection'

function mapArticle(article: any): Article {
  return new Article({
    id: String(article.id),
    sectionCode: article.section_code,
    subsectionCode: article.subsection_code,
    title: article.title,
    author: article.author,
    content: article.content,
    externalLink: article.external_link,
    tags: article.tags,
    pos: article.pos || 0,
    createdAt: article.created_at,
    updatedAt: article.updated_at,
  })
}

function mapSubsection(subsection: any): Subsection {
  return new Subsection({
    id: String(subsection.id),
    sectionCode: subsection.section_code,
    title: subsection.title,
    code: subsection.code,
    articles: subsection.articles?.map(mapArticle) || [],
  })
}

function mapSection(section: any): Section {
  return new Section({
    id: String(section.id),
    moduleCode: section.module_code,
    title: section.title,
    code: section.code,
    subsections: section.subsections?.map(mapSubsection) || [],
    articles: section.articles?.map(mapArticle) || [],
  })
}

// fetchModuleDetail 获取模块详情
export async function fetchModuleDetail(moduleCode: string): Promise<Module> {
  const { payload } = await http.get<{ module_detail: any }>(`/blog/moduleDetail?module_code=${moduleCode}`)
  return new Module({
    ...payload.module_detail,
    id: String(payload.module_detail.id),
    sections: payload.module_detail.sections?.map(mapSection) || [],
  })
}

// fetchArticleDetail 获取文章详情
export async function fetchArticleDetail(articleID: string): Promise<Article> {
  const { payload } = await http.get<{ article_detail: any }>(`/blog/articleDetail?article_id=${articleID}`)
  return mapArticle(payload.article_detail)
}
