import type { Section } from '../types/section'
import { fetchArticlesBySectionCode } from './article'

// mock 数据
const mockSections: Section[] = [
  {
    id: 1,
    moduleCode: 'ai',
    title: 'AI 发展史',
    code: 'ai_history',
    articles: [],
  },
  {
    id: 2,
    moduleCode: 'ai',
    title: '理解图灵',
    code: 'turing',
    articles: [],
  },
  {
    id: 3,
    moduleCode: 'ai',
    title: 'Prompt 工程',
    code: 'prompt',
    articles: [],
  },
  {
    id: 4,
    moduleCode: 'golang',
    title: 'golang 基础',
    code: 'golang_basic',
    articles: [],
  },
  {
    id: 5,
    moduleCode: 'golang',
    title: '并发编程',
    code: 'concurrency',
    articles: [],
  },
  {
    id: 6,
    moduleCode: 'golang',
    title: 'golang 中的设计模式',
    code: 'design_pattern',
    articles: [],
  },
  {
    id: 7,
    moduleCode: 'architecture',
    title: '设计模式',
    code: 'design_pattern',
    articles: [],
  },
  {
    id: 8,
    moduleCode: 'architecture',
    title: '架构设计',
    code: 'architecture_design',
    articles: [],
  },
  {
    id: 9,
    moduleCode: 'programming',
    title: '重构',
    code: 'refactoring',
    articles: [],
  },
  {
    id: 10,
    moduleCode: 'programming',
    title: '实现模型',
    code: 'implementation_model',
    articles: [],
  },
]

// fetchSectionsByModuleCode 获取模块下的所有分类
export async function fetchSectionsByModuleCode(moduleCode: string): Promise<Section[]> {
  return new Promise(resolve => setTimeout(async () => {
    console.log(`in fetchSectionsByModuleCode, moduleCode: ${moduleCode}`)

    // 先过滤出匹配的 sections
    let fetchedSections = mockSections.filter(s => s.moduleCode === moduleCode)

    // 然后异步加载每个 section 的文章
    for (let section of fetchedSections) {
      if (section.articles && section.articles.length === 0) {
        section.articles = await fetchArticlesBySectionCode(section.code)
      }
    }

    console.log(`fetchedSections: ${JSON.stringify(fetchedSections)}`)

    resolve(fetchedSections)
  }, 300))
}

// fetchSectionByCode 获取模块详情
export async function fetchSectionByCode(code: string): Promise<Section | undefined> {
  return new Promise(resolve => setTimeout(() => resolve(mockSections.find(m => m.code === code)), 300))
}