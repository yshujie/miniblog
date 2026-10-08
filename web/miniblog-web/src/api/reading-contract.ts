import { Article } from '@/types/article'
import { Module } from '@/types/module'
import { Section } from '@/types/section'
import { Subsection } from '@/types/subsection'

export interface ModuleSummaryDTO { id?: string | number | null; code: string; title: string }
export interface ArticleDTO {
  id: string; title: string; module_code?: string; section_code?: string; subsection_code?: string
  external_link?: string; author?: string; content?: string; tags?: string[] | null
  pos?: number; created_at?: string; updated_at?: string
}
export interface SubsectionDTO {
  id: string; code: string; title: string; section_code: string; articles?: ArticleDTO[] | null
}
export interface SectionDTO {
  id: string; code: string; title: string; module_code: string
  articles?: ArticleDTO[] | null; subsections?: SubsectionDTO[] | null
}
export interface ModuleDetailDTO extends ModuleSummaryDTO { id: string | number; sections?: SectionDTO[] | null }

export function decimalID(value: string | number): string {
  if (typeof value === 'number' && !Number.isSafeInteger(value)) {
    throw new Error('服务器返回的 ID 无法准确读取')
  }
  const id = String(value)
  if (!/^\d{1,19}$/.test(id) || BigInt(id) < 1n || BigInt(id) > 9223372036854775807n) {
    throw new Error('文章 ID 无效')
  }
  return BigInt(id).toString()
}

export function mapArticle(data: ArticleDTO, moduleCode = ''): Article {
  return new Article({
    id: decimalID(data.id), title: data.title, moduleCode: data.module_code || moduleCode,
    sectionCode: data.section_code, subsectionCode: data.subsection_code,
    externalLink: data.external_link, author: data.author, content: data.content,
    tags: data.tags ?? [], pos: data.pos, createdAt: data.created_at, updatedAt: data.updated_at,
  })
}

export function mapModuleSummary(data: ModuleSummaryDTO): Module {
  // Legacy summaries emit id=0. Modules are located by code; only articles require a positive ID.
  const id = data.id == null || data.id === '' || data.id === 0 || data.id === '0' ? '' : decimalID(data.id)
  return new Module({ id, code: data.code, title: data.title })
}

export function mapModuleDetail(data: ModuleDetailDTO): Module {
  return new Module({
    id: decimalID(data.id), code: data.code, title: data.title,
    sections: (data.sections ?? []).map(section => new Section({
      id: decimalID(section.id), code: section.code, title: section.title,
      moduleCode: section.module_code || data.code,
      articles: (section.articles ?? []).map(article => mapArticle(article, data.code)),
      subsections: (section.subsections ?? []).map(subsection => new Subsection({
        id: decimalID(subsection.id), code: subsection.code, title: subsection.title,
        sectionCode: subsection.section_code || section.code,
        articles: (subsection.articles ?? []).map(article => mapArticle(article, data.code)),
      })),
    })),
  })
}
