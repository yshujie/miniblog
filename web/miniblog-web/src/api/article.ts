import type { Article } from '../types/article'

// mock 数据
const mockArticles: Article[] = [
  {
    id: 1,
    sectionCode: 'ai_history',
    title: 'AI 的起源：达特茅斯会议',
    summary: '达特茅斯会议是人工智能领域的里程碑事件，它标志着人工智能作为一门独立学科的诞生。',
    content: '<p>达特茅斯会议是人工智能领域的里程碑事件，它标志着人工智能作为一门独立学科的诞生。</p>',
    author: 'clack',
    createdAt: '2025-05-01',
    updatedAt: '2025-05-01',
    tags: ['ai', '达特茅斯会议']
  },
  {
    id: 2,
    sectionCode: 'turing',
    title: 'AI 的定义：理解图灵',
    summary: '图灵测试是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    content: '<p>图灵测试是人工智能领域的一个重要概念，它定义了人工智能的本质。</p>',
    author: 'clack',
    createdAt: '2025-05-02',
    updatedAt: '2025-05-02',
    tags: ['ai', '图灵测试']
  },
  {
    id: 3,
    sectionCode: 'prompt',
    title: '信息论与AI',
    summary: '信息论是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    content: '<p>信息论是人工智能领域的一个重要概念，它定义了人工智能的本质。</p>',
    author: 'clack',
    createdAt: '2025-05-03',
    updatedAt: '2025-05-03',
    tags: ['ai', '信息论']
  },
  {
    id: 4,
    sectionCode: 'golang_basic',
    title: 'AI 的定义：理解图灵',
    summary: '图灵测试是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    content: '<p>图灵测试是人工智能领域的一个重要概念，它定义了人工智能的本质。</p>',
    author: 'clack',
    createdAt: '2025-05-02',
    updatedAt: '2025-05-02',
    tags: ['ai', '图灵测试']
  },
  {
    id: 5,
    sectionCode: 'concurrency',
    title: 'Go 语言的并发模型',
    summary: 'Go 语言的并发模型是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    content: '<p>Go 语言的并发模型是人工智能领域的一个重要概念，它定义了人工智能的本质。</p>',
    author: 'clack',
    createdAt: '2025-05-02',
    updatedAt: '2025-05-02',
    tags: ['golang', '并发模型']
  },
  {
    id: 6,
    sectionCode: 'golang_basic',
    title: 'Go 中的值类型与引用类型',
    summary: 'Go 中的值类型与引用类型是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    content: '<p>Go 中的值类型与引用类型是人工智能领域的一个重要概念，它定义了人工智能的本质。</p>',
    author: 'clack',
    createdAt: '2025-05-02',
    updatedAt: '2025-05-02',
    tags: ['golang', '值类型', '引用类型']
  },
  {
    id: 7,
    sectionCode: 'design_pattern',
    title: '架构设计',
    summary: '架构设计是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    content: '<p>架构设计是人工智能领域的一个重要概念，它定义了人工智能的本质。</p>',
    author: 'clack',
    createdAt: '2025-05-02',
    updatedAt: '2025-05-02',
    tags: ['架构设计']
  },
  {
    id: 8,
    sectionCode: 'architecture',
    title: 'DDD 领域驱动设计',
    summary: 'DDD 领域驱动设计是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    content: '<p>DDD 领域驱动设计是人工智能领域的一个重要概念，它定义了人工智能的本质。</p>',
    author: 'clack',
    createdAt: '2025-05-02',
    updatedAt: '2025-05-02',
    tags: ['架构设计', 'DDD']
  },
  {
    id: 9,
    sectionCode: 'architecture',
    title: '六边形架构',
    summary: '六边形架构是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    content: '<p>六边形架构是人工智能领域的一个重要概念，它定义了人工智能的本质。</p>',
    author: 'clack',
    createdAt: '2025-05-02',
    updatedAt: '2025-05-02',
    tags: ['架构设计', '六边形架构']
  },
  {
    id: 10,
    sectionCode: 'architecture',
    title: '设计原则：单一职责原则',
    summary: '单一职责原则是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    content: '<p>单一职责原则是人工智能领域的一个重要概念，它定义了人工智能的本质。</p>',
    author: 'clack',
    createdAt: '2025-05-02',
    updatedAt: '2025-05-02',
    tags: ['编程', '单一职责原则']
  },
  {
    id: 11,
    sectionCode: 'architecture',
    title: '设计原则：开闭原则',
    summary: '开闭原则是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    content: '<p>开闭原则是人工智能领域的一个重要概念，它定义了人工智能的本质。</p>',
    author: 'clack',
    createdAt: '2025-05-02',
    updatedAt: '2025-05-02',
    tags: ['编程', '开闭原则']
  },
  {
    id: 12,
    sectionCode: 'architecture',
    title: '设计原则：里氏替换原则',
    summary: '里氏替换原则是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    content: '<p>里氏替换原则是人工智能领域的一个重要概念，它定义了人工智能的本质。</p>',
    author: 'clack',
    createdAt: '2025-05-02',
    updatedAt: '2025-05-02',
    tags: ['编程', '里氏替换原则']
  },
  {
    id: 13,
    sectionCode: 'architecture',
    title: '设计原则：依赖倒置原则',
    summary: '依赖倒置原则是人工智能领域的一个重要概念，它定义了人工智能的本质。',
    content: '<p>依赖倒置原则是人工智能领域的一个重要概念，它定义了人工智能的本质。</p>',
    author: 'clack',
    createdAt: '2025-05-02',
    updatedAt: '2025-05-02',
    tags: ['编程', '依赖倒置原则']
  }
]

// fetchArticlesBySectionCode 获取分类文章列表
export async function fetchArticlesBySectionCode(sectionCode: string): Promise<Article[]> {
  return new Promise(resolve => setTimeout(() => {
    console.log(`in fetchArticlesBySectionCode, sectionCode: ${sectionCode}`)

    // 查找 articles 数据
    var fetchedArticles = mockArticles.filter(a => a.sectionCode === sectionCode)

    resolve(fetchedArticles)
  }, 300))
}

// fetchArticleById 获取文章详情
export async function fetchArticleById(id: number): Promise<Article | undefined> {
  return new Promise(resolve => setTimeout(async () => {
    console.log(`in fetchArticleById, id: ${id}`)

    resolve(mockArticles.find(a => a.id === id))
  }, 300))
}