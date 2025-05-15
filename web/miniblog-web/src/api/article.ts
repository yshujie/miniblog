import { Article } from '../types/article'

// mock 数据
const mockArticles: Article[] = [
  {
    id: 1,
    title: '第一篇博客',
    summary: '这是一篇关于 Vue3 博客系统的文章摘要',
    content: '<p>欢迎使用 GoodBlog Vue3。你可以开始编写你的第一篇博客了！</p>',
    author: 'koala',
    createdAt: '2025-05-01',
    tags: ['Vue3', '博客']
  },
  {
    id: 2,
    title: '第二篇博客',
    summary: '本篇介绍如何用 TypeScript 和 Vue3 结合开发。',
    content: '<p>TypeScript 让 Vue3 项目更健壮可维护。</p>',
    author: 'koala',
    createdAt: '2025-05-02',
    tags: ['TypeScript']
  }
]

export async function fetchArticleList(): Promise<Article[]> {
  return new Promise(resolve => setTimeout(() => resolve(mockArticles), 300))
}

export async function fetchArticleDetail(id: number): Promise<Article | undefined> {
  return new Promise(resolve =>
    setTimeout(() => resolve(mockArticles.find(a => a.id === id)), 300)
  )
}
