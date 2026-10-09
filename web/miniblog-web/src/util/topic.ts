const descriptions: Record<string, string> = {
  go: '从语言基础到运行机制，在代码中理解 Go。',
  ddd: '从需求出发，把业务知识放进清晰的模型与边界。',
  project: '记录真实项目里的分析、选择和实现过程。',
  ai: '记录对人工智能的探索，把问题与理解连接起来。',
  refactor: '从代码中的问题出发，持续改善结构与表达。',
  database: '理解数据的组织、存储与访问方式。',
}
export const topicDescription = (code: string) => descriptions[code] || '按章节探索这个主题，在阅读中积累理解。'
