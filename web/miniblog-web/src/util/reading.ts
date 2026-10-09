import type { Module } from '@/types/module'
import type { Article } from '@/types/article'

export function firstArticle(module: Module): Article | undefined {
  for (const section of module.sections) {
    for (const subsection of section.subsections) {
      if (subsection.articles.length) return subsection.articles[0]
    }
    if (section.articles.length) return section.articles[0]
  }
}

export function containsPlacement(module: Module, article: Article): boolean {
  const section = module.sections.find(item => item.code === article.sectionCode)
  if (!section) return false
  const articles = article.subsectionCode
    ? section.subsections.find(item => item.code === article.subsectionCode)?.articles
    : section.articles
  return articles?.some(item => item.id === article.id) ?? false
}

export function safeExternalURL(value: string): string | null {
  try {
    const url = new URL(value)
    return ['http:', 'https:'].includes(url.protocol) && !url.username && !url.password ? url.href : null
  } catch { return null }
}

export function articleTitle(article: Article): string {
  return article.title?.trim() || (safeExternalURL(article.externalLink) ? new URL(article.externalLink).hostname : '未命名文章')
}

// An ordinary Notion public page denies third-party frames. Keep the current
// original link, and derive its embed from the same page rather than a legacy URL.
export function articleReadingLinks(article: Pick<Article, 'readingURL' | 'externalLink'>): { originalURL: string | null; embedURL: string | null } {
  const originalURL = safeExternalURL(article.readingURL) || safeExternalURL(article.externalLink)
  if (!originalURL) return { originalURL: null, embedURL: null }
  const url = new URL(originalURL)
  const notionHost = url.hostname === 'notion.site' || url.hostname.endsWith('.notion.site') ||
    url.hostname === 'notion.so' || url.hostname.endsWith('.notion.so')
  // Database/view selectors can address another page; never infer it from the
  // path ID. Unknown slugs, custom domains and non-Notion links stay unchanged.
  if (!notionHost || url.pathname.startsWith('/ebd/') || url.searchParams.has('p') || url.searchParams.has('v')) {
    return { originalURL, embedURL: originalURL }
  }
  const segment = url.pathname.replace(/\/+$/, '').split('/').pop() || ''
  const match = /(?:^|-)([0-9a-f]{32}|[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$/i.exec(segment)
  const pageID = match?.[1]?.replace(/-/g, '').toLowerCase()
  return { originalURL, embedURL: pageID ? `${url.origin}/ebd/${pageID}` : originalURL }
}
