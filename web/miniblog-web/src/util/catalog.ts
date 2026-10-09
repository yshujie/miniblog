import type { Article } from '@/types/article'
import type { Module } from '@/types/module'
import type { Section } from '@/types/section'
import type { Subsection } from '@/types/subsection'

export interface CatalogEntry {
  article: Article
  section: Section
  subsection?: Subsection
}

export function flattenCatalog(module: Pick<Module, 'sections'>): CatalogEntry[] {
  return module.sections.flatMap(section => [
    ...section.subsections.flatMap(subsection => subsection.articles.map(article => ({ article, section, subsection }))),
    ...section.articles.map(article => ({ article, section })),
  ])
}

export function filterCatalog(entries: readonly CatalogEntry[], query: string): CatalogEntry[] {
  const keyword = query.trim().toLowerCase()
  return keyword
    ? entries.filter(entry => (entry.article.title || '').trim().toLowerCase().includes(keyword))
    : [...entries]
}

export function findCatalogEntry(entries: readonly CatalogEntry[], articleId: string): CatalogEntry | undefined {
  return entries.find(entry => entry.article.id === articleId)
}

export function catalogNeighbors(entries: readonly CatalogEntry[], articleId: string): {
  previous?: CatalogEntry
  next?: CatalogEntry
} {
  const current = entries.findIndex(entry => entry.article.id === articleId)
  if (current < 0) return { previous: undefined, next: undefined }
  return { previous: entries[current - 1], next: entries[current + 1] }
}
